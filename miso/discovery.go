package miso

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/curtisnewbie/miso/util"
)

var (
	_ ServiceRegistry = ServerListServiceRegistry{}
	_ ServiceRegistry = hardcodedServiceRegistry{}
)

var (
	ErrMissingServiceName      = errors.New("service name is required")
	ErrServiceInstanceNotFound = errors.New("unable to find any available service instance")
	ErrServerListNotFound      = errors.New("fail to find ServerList implemnetation")
)

var discModule = InitAppModuleFunc(newModule)

func init() {
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Bootstrap Service Discovery",
		Condition: func(rail Rail) (bool, error) { return true, nil },
		Bootstrap: func(rail Rail) error {
			m := discModule()
			sl := m.getServerList()
			if sl == nil {
				return nil
			}
			for _, s := range App().Config().GetPropStrSlice(PropSDSubscrbe) {
				if err := sl.Subscribe(rail, s); err != nil {
					rail.Warnf("Failed to subscrbe %v, %v", s, err)
				}
			}
			return nil
		},
		Order: 5, // default is 0, runs after all default bootstrap components
	})
}

type discoveryModule struct {
	// Property based ServiceRegistry
	propBasedServiceRegistry hardcodedServiceRegistry

	// ServerList based ServiceRegistry
	//
	// Server selection can be customized by replacing the Rule.
	dynamicServiceRegistry ServerListServiceRegistry

	// Map of ServerChangeListeners
	serverChangeListeners ServerChangeListenerMap

	// Get ServerList implementation
	getServerList func() ServerList
}

func newModule() *discoveryModule {
	return &discoveryModule{
		propBasedServiceRegistry: hardcodedServiceRegistry{},
		dynamicServiceRegistry:   ServerListServiceRegistry{Rule: RandomServerSelector},
		serverChangeListeners: ServerChangeListenerMap{
			Listeners: map[string][]func(){},
			Pool:      util.NewCpuAsyncPool(),
		},
		getServerList: func() ServerList { return nil },
	}
}

func (m *discoveryModule) changeGetServerList(f func() ServerList) {
	if f == nil {
		panic("getServerList(..) cannot be nil")
	}
	m.getServerList = f
}

func (m *discoveryModule) selectAnyServer(rail Rail, name string) (Server, error) {
	return m.selectServer(rail, name, RandomServerSelector)
}

func (m *discoveryModule) selectServer(rail Rail, name string, selector func(servers []Server) int) (Server, error) {
	serverList := m.getServerList()
	var servers []Server
	if serverList != nil {
		servers = serverList.ListServers(rail, name)
		if len(servers) < 1 {
			if !serverList.IsSubscribed(rail, name) {
				if err := serverList.Subscribe(rail, name); err != nil {
					return Server{}, fmt.Errorf("failed to subscribe service, service not avaliable, %w", err)
				}
				if err := serverList.PollInstance(rail, name); err != nil {
					return Server{}, fmt.Errorf("failed to poll service instance, service not available, %w", err)
				}
				return m.selectServer(rail, name, selector)
			}
		}
	}
	if len(servers) < 1 {
		servers, _ = m.propBasedServiceRegistry.ListServers(rail, name)
	}
	if len(servers) < 1 {
		return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrServiceInstanceNotFound)
	}
	selected := selector(servers)
	if selected >= 0 && selected < len(servers) {
		return servers[selected], nil
	}
	return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrServiceInstanceNotFound)
}

func (m *discoveryModule) getServiceRegistry() ServiceRegistry {
	return m.dynamicServiceRegistry
}

func (m *discoveryModule) subscribeServerChanges(rail Rail, name string, cbk func()) error {
	sl := m.getServerList()
	if sl == nil {
		return ErrServerListNotFound
	}
	if err := sl.Subscribe(rail, name); err != nil {
		return fmt.Errorf("failed to subscribe to service %v, %w", name, err)
	}
	m.serverChangeListeners.SubscribeChange(name, cbk)
	return nil
}

func (m *discoveryModule) triggerServerChangeListeners(service string) {
	m.serverChangeListeners.TriggerListeners(service)
}

type ServerList interface {
	PollInstances(rail Rail) error
	PollInstance(rail Rail, name string) error
	ListServers(rail Rail, name string) []Server
	IsSubscribed(rail Rail, service string) bool
	Subscribe(rail Rail, service string) error
	Unsubscribe(rail Rail, service string) error
	UnsubscribeAll(rail Rail) error
}

// Server selector, returns index of the selected one.
type ServerSelector func(servers []Server) int

type Server struct {
	Protocol string
	Address  string
	Port     int
	Meta     map[string]string
}

// Build the complete request url.
func (c *Server) BuildUrl(relUrl string) string {
	if !strings.HasPrefix(relUrl, "/") {
		relUrl = "/" + relUrl
	}
	if c.Protocol == "" {
		c.Protocol = "http://"
	}
	return c.Protocol + c.ServerAddress() + relUrl
}

// Build server address with host and port concatenated, e.g., 'localhost:8080'
func (c *Server) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

type ServiceRegistry interface {
	ResolveUrl(rail Rail, service string, relativeUrl string) (string, error)
	ListServers(rail Rail, service string) ([]Server, error)
}

// Get service registry.
//
// Service registry initialization is lazy, don't store the retunred value in global var.
func GetServiceRegistry() ServiceRegistry {
	return discModule().getServiceRegistry()
}

// Service registry backed by loaded configuration.
type hardcodedServiceRegistry struct {
}

func (r hardcodedServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {
	if util.IsBlankStr(service) {
		return "", ErrMissingServiceName
	}

	host := r.serverHostFromProp(service)
	port := r.serverPortFromProp(service)

	if util.IsBlankStr(host) {
		return httpProto + service + relativeUrl, nil
	}

	return httpProto + fmt.Sprintf("%s:%d", host, port) + relativeUrl, nil
}

func (r hardcodedServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	if util.IsBlankStr(service) {
		return []Server{}, ErrMissingServiceName
	}

	host := r.serverHostFromProp(service)
	port := r.serverPortFromProp(service)

	if !util.IsBlankStr(host) {
		return []Server{{Address: host, Port: port, Meta: map[string]string{}}}, nil
	}

	return []Server{{Address: service, Port: port, Meta: map[string]string{}}}, nil
}

func (r hardcodedServiceRegistry) serverHostFromProp(name string) string {
	if name == "" {
		return ""
	}
	return GetPropStr("client.addr." + name + ".host")
}

func (r hardcodedServiceRegistry) serverPortFromProp(name string) int {
	if name == "" {
		return 0
	}
	return GetPropInt("client.addr." + name + ".port")
}

type ServerChangeListenerMap struct {
	Listeners map[string][]func()
	Pool      *util.AsyncPool
	sync.RWMutex
}

func (s *ServerChangeListenerMap) TriggerListeners(name string) {
	s.RLock()
	defer s.RUnlock()
	if listeners, ok := s.Listeners[name]; ok {
		for i := range listeners {
			s.Pool.Go(listeners[i])
		}
	}
}

func (s *ServerChangeListenerMap) SubscribeChange(name string, cbk func()) {
	s.Lock()
	defer s.Unlock()

	if v, ok := s.Listeners[name]; ok {
		v = append(v, cbk)
		s.Listeners[name] = v
	} else {
		s.Listeners[name] = []func(){cbk}
	}
}

type ServerListServiceRegistry struct {
	Rule ServerSelector
}

func (c ServerListServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {
	m := discModule()
	server, err := m.selectServer(rail, service, c.Rule)
	if err != nil {
		return "", err
	}

	return server.BuildUrl(relativeUrl), nil
}

func (c ServerListServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	m := discModule()
	sl := m.getServerList()
	if sl == nil {
		return nil, ErrServerListNotFound
	}
	servers := sl.ListServers(rail, service)
	if len(servers) < 1 {
		return m.propBasedServiceRegistry.ListServers(rail, service)
	}
	return servers, nil
}

// Subscribe to changes to service instances.
func SubscribeServerChanges(rail Rail, name string, cbk func()) error {
	return discModule().subscribeServerChanges(rail, name, cbk)
}

// Trigger service changes listeners.
func TriggerServerChangeListeners(service string) {
	discModule().triggerServerChangeListeners(service)
}

// Select one Server based on the provided selector algorithm.
func SelectServer(rail Rail, name string, selector func(servers []Server) int) (Server, error) {
	return discModule().selectServer(rail, name, selector)
}

// Select one Server randomly.
func SelectAnyServer(rail Rail, name string) (Server, error) {
	return discModule().selectAnyServer(rail, name)
}

// Select Server randomly.
func RandomServerSelector(servers []Server) int {
	if len(servers) < 1 {
		return -1
	}
	return rand.Int() % len(servers)
}

// Get ServerList, may return nil.
func GetServerList() ServerList {
	return discModule().getServerList()
}

// Change GetServiceList implmentation.
func ChangeGetServerList(f func() ServerList) {
	discModule().changeGetServerList(f)
}
