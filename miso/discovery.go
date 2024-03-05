package miso

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

var (
	// Select Server randomly.
	RandomServerSelector ServerSelector = func(servers []Server) int {
		return rand.Int() % len(servers)
	}

	// Property based ServiceRegistry
	PropBasedServiceRegistry = HardcodedServiceRegistry{}

	// ServerList based ServiceRegistry
	//
	// Server selection can be customized by replacing the Rule.
	DynamicServiceRegistry = ServerListServiceRegistry{Rule: RandomServerSelector}

	ErrMissingServiceName      = errors.New("service name is required")
	ErrServiceInstanceNotFound = errors.New("unable to find any available service instance")
	ErrServerListNotFound      = errors.New("fail to find ServerList implemnetation")

	// ServiceRegistry that is currently in use.
	clientServiceRegistry ServiceRegistry = nil

	// Map of ServerChangeListeners
	serverChangeListeners = ServerChangeListenerMap{
		Listeners: map[string][]func(){},
		Pool:      NewAsyncPool(500, 4),
	}

	// Get ServerList implementation
	GetServerList func() ServerList
)

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
	if clientServiceRegistry != nil {
		return clientServiceRegistry
	}
	return PropBasedServiceRegistry
}

// Service registry backed by loaded configuration.
type HardcodedServiceRegistry struct {
}

func (r HardcodedServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {
	if IsBlankStr(service) {
		return "", ErrMissingServiceName
	}

	host := r.serverHostFromProp(service)
	port := r.serverPortFromProp(service)

	if IsBlankStr(host) {
		return httpProto + service + relativeUrl, nil
	}

	return httpProto + fmt.Sprintf("%s:%d", host, port) + relativeUrl, nil
}

func (r HardcodedServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	if IsBlankStr(service) {
		return []Server{}, ErrMissingServiceName
	}

	host := r.serverHostFromProp(service)
	port := r.serverPortFromProp(service)

	if IsBlankStr(host) {
		return []Server{{Address: host, Port: port, Meta: map[string]string{}}}, nil
	}

	return []Server{{Address: service, Port: 0, Meta: map[string]string{}}}, nil
}

func (r HardcodedServiceRegistry) serverHostFromProp(name string) string {
	if name == "" {
		return ""
	}
	return GetPropStr("client.addr." + name + ".host")
}

func (r HardcodedServiceRegistry) serverPortFromProp(name string) int {
	if name == "" {
		return 0
	}
	return GetPropInt("client.addr." + name + ".port")
}

type ServerChangeListenerMap struct {
	Listeners map[string][]func()
	Pool      *AsyncPool
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

// Select one Server based on the provided selector.
//
// GetServerList() is internally called to obtain current ServerList implementation.
//
// If none is found and the service is not subscribed yet in the ServerList, this func subscribes to the service and polls the service instances immediately.
//
// If ServerList indeed doesn't find any available instance for the service, ErrServiceInstanceNotFound is returned.
func SelectServer(rail Rail, name string, selector func(servers []Server) int) (Server, error) {
	serverList := GetServerList()
	if serverList == nil {
		return Server{}, ErrServerListNotFound
	}
	servers := serverList.ListServers(rail, name)
	if len(servers) < 1 {
		if !serverList.IsSubscribed(rail, name) {
			if err := serverList.Subscribe(rail, name); err != nil {
				return Server{}, fmt.Errorf("failed to subscribe service, service not avaliable, %w", err)
			}
			if err := serverList.PollInstance(rail, name); err != nil {
				return Server{}, fmt.Errorf("failed to poll service instance, service not available, %w", err)
			}
			return SelectServer(rail, name, selector)
		}
		return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrServiceInstanceNotFound)
	}
	selected := selector(servers)
	if selected >= 0 && selected < len(servers) {
		return servers[selected], nil
	}
	return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrServiceInstanceNotFound)
}

// Select one Server randomly.
//
// This func internally calls SelectServer with RandomServerSelector.
func SelectAnyServer(rail Rail, name string) (Server, error) {
	return SelectServer(rail, name, RandomServerSelector)
}

type ServerListServiceRegistry struct {
	Rule ServerSelector
}

func (c ServerListServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {
	server, err := SelectServer(rail, service, c.Rule)
	if err != nil {
		return "", err
	}

	return server.BuildUrl(relativeUrl), nil
}

func (c ServerListServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	sl := GetServerList()
	if sl == nil {
		return nil, ErrServerListNotFound
	}
	return sl.ListServers(rail, service), nil
}

// Subscribe to changes to service instances.
//
// Callback is triggered asynchronously.
func SubscribeServerChanges(rail Rail, name string, cbk func()) error {
	sl := GetServerList()
	if sl == nil {
		return ErrServerListNotFound
	}
	if err := sl.Subscribe(rail, name); err != nil {
		return fmt.Errorf("failed to subscribe to service %v, %w", name, err)
	}
	serverChangeListeners.SubscribeChange(name, cbk)
	return nil
}
