//go:build !excl_consul
// +build !excl_consul

package miso

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

const (
	// Service registration status - passing.
	ConsulRegiStatusPassing = "passing"

	// Zero value for empty serviceId
	ServiceIdNil = "nil"
)

var (
	// Service registration
	regSub = &serviceRegistration{serviceId: ServiceIdNil}

	// Global handle to the Consul client
	consulp = &consulHolder{consul: nil}

	// Holder (cache) of service list and their instances
	serviceListHolder = &ServiceListHolder{Instances: map[string][]ConsulServer{}}

	// server list polling subscription
	serverListPSub = &serverListPollingSubscription{sub: nil}

	// server instance not found
	ErrConsulServiceInstanceNotFound error = errors.New("unable to find any available service instance")

	// Api for Consul
	ConsulApi = ConsulApiImpl{}

	// Select ConsulServer randomly.
	RandomConsulServerSelector = func(servers []ConsulServer) int { return rand.Int() % len(servers) }
)

func init() {
	SetDefProp(PropConsulEnabled, false)
	SetDefProp(PropConsulAddress, "localhost:8500")
	SetDefProp(PropConsulHealthcheckUrl, "/health")
	SetDefProp(PropConsulHealthCheckInterval, "15s")
	SetDefProp(PropConsulHealthcheckTimeout, "3s")
	SetDefProp(PropConsulHealthCheckFailedDeregAfter, "120s")
	SetDefProp(PropConsulRegisterDefaultHealthcheck, true)
	SetDefProp(PropConsulFetchServerInterval, 15)

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Boostrap Consul",
		Bootstrap: ConsulBootstrap,
		Condition: ConsulBootstrapCondition,
		Order:     -10,
	})
}

type ConsulApiImpl struct{}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func (c ConsulApiImpl) CatalogFetchServiceNodes(rail Rail, name string) ([]*api.CatalogService, error) {
	defer DebugTimeOp(rail, time.Now(), "CatalogFetchServiceNodes")
	client, err := GetConsulClient()
	if err != nil {
		return nil, err
	}

	services, _, err := client.Catalog().Service(name, "", nil)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// Fetch all registered services, this method always call Consul instead of reading from cache
func (c ConsulApiImpl) CatalogFetchServiceNames(rail Rail) (map[string][]string, error) {
	client, e := GetConsulClient()
	if e != nil {
		return nil, e
	}
	services, _, err := client.Catalog().Services(nil)
	rail.Debugf("CatalogFetchServiceNames, %+v, %v", services, err)
	return services, err
}

func (c ConsulApiImpl) DeregisterService(serviceId string) error {
	client, err := GetConsulClient()
	if err != nil {
		return fmt.Errorf("failed to get consul client, %v", err)
	}
	return client.Agent().ServiceDeregister(serviceId)
}

type serverListPollingSubscription struct {
	sub *time.Ticker
	mu  sync.Mutex
}

type consulHolder struct {
	consul *api.Client
	mu     sync.RWMutex
}

type serviceRegistration struct {
	serviceId string
	mu        sync.Mutex
}

// Holder of a list of ServiceHolder
type ServiceListHolder struct {
	sync.RWMutex
	Instances map[string][]ConsulServer
}

type ConsulServer struct {
	Protocol string
	Address  string
	Port     int
	Meta     map[string]string
}

func (c *ConsulServer) BuildUrl(relUrl string) string {
	if !strings.HasPrefix(relUrl, "/") {
		relUrl = "/" + relUrl
	}
	if c.Protocol == "" {
		c.Protocol = "http://"
	}
	return c.Protocol + c.ServerAddress() + relUrl
}

func (c *ConsulServer) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

// Subscribe to server list, refresh server list every 30s
func SubscribeServerList(everyNSec int) {
	serverListPSub.mu.Lock()
	defer serverListPSub.mu.Unlock()

	if serverListPSub.sub != nil {
		return
	}

	serverListPSub.sub = time.NewTicker(time.Duration(everyNSec) * time.Second)
	c := serverListPSub.sub.C
	go func() {
		rail := EmptyRail()
		for {
			PollServiceListInstances(rail)
			<-c
		}
	}()
}

// stop refreshing server list
func UnsubscribeServerList() {
	serverListPSub.mu.Lock()
	defer serverListPSub.mu.Unlock()

	if serverListPSub.sub == nil {
		return
	}

	serverListPSub.sub.Stop()
}

/*
Check if consul is enabled

This func looks for following prop:

	"consul.enabled"
*/
func IsConsulEnabled() bool {
	return GetPropBool(PropConsulEnabled)
}

// Poll all service list and cache them.
func PollServiceListInstances(rail Rail) {
	names, err := ConsulApi.CatalogFetchServiceNames(rail)
	if err != nil {
		rail.Errorf("Failed to CatalogFetchServiceNames, %v", err)
		return
	}

	for name := range names {
		err := fetchAndCacheServiceNodes(rail, name)
		if err != nil {
			rail.Warnf("Failed to poll service service for '%s', err: %v", name, err)
		}
	}
}

// Fetch and cache services nodes.
func fetchAndCacheServiceNodes(rail Rail, name string) error {
	serviceListHolder.Lock()
	defer serviceListHolder.Unlock()

	services, err := ConsulApi.CatalogFetchServiceNodes(rail, name)
	if err != nil {
		return fmt.Errorf("failed to FetchServicesByName, name: %v, %v", name, err)
	}
	servers := make([]ConsulServer, 0, len(services))
	for i := range services {
		s := services[i]
		servers = append(servers, ConsulServer{
			Meta:    s.ServiceMeta,
			Address: s.ServiceAddress,
			Port:    s.ServicePort,
		})
	}
	rail.Debugf("Fetched nodes for service: %v, %+v", name, servers)
	serviceListHolder.Instances[name] = servers
	return err
}

/*
Resolve request url for the given service.

The resolved url will be in format: "http://" + host + ":" + port + "/" + relUrl.

Return ErrServiceInstanceNotFound if no instance is found.
*/
func ConsulResolveRequestUrl(serviceName string, relUrl string) (string, error) {
	server, err := SelectConsulServer(serviceName, RandomConsulServerSelector)
	if err != nil {
		return "", err
	}
	return server.BuildUrl(relUrl), nil
}

/*
Resolve service address (host:port)

This func will first read the cache, trying to resolve the services address
without actually requesting consul, and only when the cache missed, it then
requests the consul

Return ErrServiceInstanceNotFound if no instance is found.
*/
func ConsulResolveServiceAddr(name string) (string, error) {
	selected, err := SelectConsulServer(name, RandomConsulServerSelector)
	if err != nil {
		return "", err
	}
	return selected.ServerAddress(), nil
}

// Select one ConsulServer based on the provided selector.
//
// If none is matched, ErrConsulServiceInstanceNotFound is returned.
func SelectConsulServer(name string, selector func(servers []ConsulServer) int) (ConsulServer, error) {
	serviceListHolder.RLock()
	defer serviceListHolder.RUnlock()
	servers := serviceListHolder.Instances[name]

	if len(servers) < 1 {
		return ConsulServer{}, fmt.Errorf("failed to select server for %v, %w", name, ErrConsulServiceInstanceNotFound)
	}
	selected := selector(servers)
	if selected >= 0 && selected < len(servers) {
		return servers[selected], nil
	}
	return ConsulServer{}, fmt.Errorf("failed to select server for %v, %w", name, ErrConsulServiceInstanceNotFound)
}

// List Consul Servers already loaded in cache.
func ListConsulServers(name string) []ConsulServer {
	serviceListHolder.RLock()
	defer serviceListHolder.RUnlock()
	servers := serviceListHolder.Instances[name]
	copied := make([]ConsulServer, 0, len(servers))
	copy(copied, servers)
	return copied
}

// Register current service
func DeregisterService() error {
	if !IsConsulClientInitialized() {
		return nil
	}

	regSub.mu.Lock()
	defer regSub.mu.Unlock()

	// not registered
	if regSub.serviceId == ServiceIdNil {
		return nil
	}

	EmptyRail().Infof("Deregistering current instance on Consul, service_id: '%s'", regSub.serviceId)

	err := ConsulApi.DeregisterService(regSub.serviceId)
	if err != nil {
		regSub.serviceId = ServiceIdNil
	}
	return err
}

/*
Register current instance as a service

If we have already registered before, current method call will be ignored.

This func looks for following prop:

	"server.port"
	"consul.registerName"
	"consul.healthCheckInterval"
	"consul.registerAddress"
	"consul.healthCheckUrl"
	"consul.healthCheckTimeout"
	"consul.healthCheckFailedDeregisterAfter"
*/
func RegisterService() error {
	var client *api.Client
	var e error

	if client, e = GetConsulClient(); e != nil {
		return e
	}

	regSub.mu.Lock()
	defer regSub.mu.Unlock()

	// registered already
	if regSub.serviceId != ServiceIdNil {
		return nil
	}

	serverPort := GetPropInt(PropServerPort)
	registerName := GetPropStr(PropConsuleRegisterName)
	if registerName == "" { // fallback to app.name
		registerName = GetPropStr(PropAppName)
	}
	registerAddress := GetPropStr(PropConsulRegisterAddress)
	healthCheckUrl := GetPropStr(PropConsulHealthcheckUrl)
	healthCheckInterval := GetPropStr(PropConsulHealthCheckInterval)
	healthCheckTimeout := GetPropStr(PropConsulHealthcheckTimeout)
	healthCheckDeregAfter := GetPropStr(PropConsulHealthCheckFailedDeregAfter)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = ResolveServerHost(GetPropStr(PropServerHost))
	}

	proposedServiceId := fmt.Sprintf("%s-%d", registerName, serverPort)
	registration := &api.AgentServiceRegistration{
		ID:      proposedServiceId,
		Name:    registerName,
		Port:    serverPort,
		Address: registerAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", registerAddress, serverPort, healthCheckUrl),
			Interval:                       healthCheckInterval,
			Timeout:                        healthCheckTimeout,
			DeregisterCriticalServiceAfter: healthCheckDeregAfter,
			Status:                         ConsulRegiStatusPassing, // for responsiveness
		},
	}

	if e = client.Agent().ServiceRegister(registration); e != nil {
		return TraceErrf(e, "failed to register on consul, registration: %+v", registration)
	}
	regSub.serviceId = proposedServiceId

	EmptyRail().Infof("Registered on Consul, serviceId: '%s'", proposedServiceId)
	return nil
}

/*
Get or init new consul client

For the first time that the consul client is initialized, this func will look for prop:

	"consul.consulAddress"
*/
func GetConsulClient() (*api.Client, error) {
	if IsConsulClientInitialized() {
		return consulp.consul, nil
	}

	consulp.mu.Lock()
	defer consulp.mu.Unlock()

	if consulp.consul != nil {
		return consulp.consul, nil
	}

	addr := GetPropStr(PropConsulAddress)
	c, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		return nil, err
	}
	consulp.consul = c
	EmptyRail().Infof("Created Consul Client on %s", addr)

	SubscribeServerList(GetPropInt(PropConsulFetchServerInterval))

	return c, nil
}

// Check whether consul client is initialized
func IsConsulClientInitialized() bool {
	consulp.mu.RLock()
	defer consulp.mu.RUnlock()
	return consulp.consul != nil
}

func ConsulBootstrap(rail Rail) error {
	// create consul client
	if _, e := GetConsulClient(); e != nil {
		return TraceErrf(e, "Failed to establish connection to Consul")
	}

	// deregister on shutdown
	AddShutdownHook(func() {
		if e := DeregisterService(); e != nil {
			rail.Errorf("Failed to deregister on Consul, %v", e)
		}
	})

	if e := RegisterService(); e != nil {
		return TraceErrf(e, "Failed to register on Consul")
	}

	ClientServiceRegistry = consulServiceRegistry{}
	rail.Debug("Using consulServiceRegistry")

	return nil
}

func ConsulBootstrapCondition(rail Rail) (bool, error) {
	return IsConsulEnabled(), nil
}

// Service registry based on Consul
type consulServiceRegistry struct {
}

func (r consulServiceRegistry) resolve(service string, relativeUrl string) (string, error) {
	return ConsulResolveRequestUrl(service, relativeUrl)
}
