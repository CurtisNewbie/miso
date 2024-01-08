//go:build !excl_consul
// +build !excl_consul

package miso

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
)

const (
	// Service registration status - passing.
	ConsulStatusPassing = "passing"

	// Zero value for empty serviceId
	ServiceIdNil = "nil"
)

var (
	// Service registration
	consulRegistration = &serviceRegistration{serviceId: ServiceIdNil}

	// Global handle to the Consul client
	consulp = &consulHolder{consul: nil}

	// Holder (cache) of service list and their instances
	consulServerList = &ServerList{servers: map[string][]Server{}, serviceWatches: map[string]*watch.Plan{}}

	// server list polling subscription
	consulServerListPoller *TickRunner = nil

	// Api for Consul.
	ConsulApi = ConsulApiImpl{}

	// Consul's implementation of ServiceRegistry.
	//
	// Customize server selection by replacing Rule.
	ConsulBasedServiceRegistry = &ConsulServiceRegistry{
		Rule: RandomServerSelector,
	}
)

type consulHolder struct {
	consul *api.Client
	mu     sync.RWMutex
}

type serviceRegistration struct {
	serviceName string
	serviceId   string
	mu          sync.Mutex
}

func init() {
	SetDefProp(PropConsulEnabled, false)
	SetDefProp(PropConsulAddress, "localhost:8500")
	SetDefProp(PropConsulHealthcheckUrl, "/health")
	SetDefProp(PropConsulHealthCheckInterval, "5s")
	SetDefProp(PropConsulHealthcheckTimeout, "3s")
	SetDefProp(PropConsulHealthCheckFailedDeregAfter, "30m")
	SetDefProp(PropConsulRegisterDefaultHealthcheck, true)
	SetDefProp(PropConsulFetchServerInterval, 30)
	SetDefProp(PropConsulDeregisterUrl, "/consul/deregister")
	SetDefProp(PropConsulEnableDeregisterUrl, false)
	SetDefProp(PropConsuleRegisterName, "${app.name}")

	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Boostrap Consul",
		Bootstrap: ConsulBootstrap,
		Condition: ConsulBootstrapCondition,
		Order:     BootstrapOrderL4,
	})
}

type ConsulApiImpl struct{}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func (c ConsulApiImpl) CatalogFetchServiceNodes(rail Rail, name string) ([]*api.CatalogService, error) {
	defer DebugTimeOp(rail, time.Now(), "CatalogFetchServiceNodes")
	client := GetConsulClient()

	services, _, err := client.Catalog().Service(name, "", nil)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// Fetch all registered services, this method always call Consul instead of reading from cache
func (c ConsulApiImpl) CatalogFetchServiceNames(rail Rail) (map[string][]string, error) {
	client := GetConsulClient()
	services, _, err := client.Catalog().Services(nil)
	rail.Debugf("CatalogFetchServiceNames, %+v, %v", services, err)
	return services, err
}

func (c ConsulApiImpl) DeregisterService(serviceId string) error {
	client := GetConsulClient()
	return client.Agent().ServiceDeregister(serviceId)
}

func (c ConsulApiImpl) RegisterService(registration *api.AgentServiceRegistration) error {
	client := GetConsulClient()
	if err := client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register consul service, registration: %+v, %w", registration, err)
	}
	return nil
}

// Holder of a list of ServiceHolder
type ServerList struct {
	sync.RWMutex
	servers        map[string][]Server
	serviceWatches map[string]*watch.Plan
}

func (s *ServerList) IsSubscribed(rail Rail, service string) bool {
	consulServerList.RLock()
	defer consulServerList.RUnlock()
	_, ok := s.serviceWatches[service]
	return ok
}

func (s *ServerList) Subscribe(rail Rail, service string) error {
	consulServerList.RLock()
	if _, ok := s.serviceWatches[service]; ok {
		consulServerList.RUnlock()
		return nil
	}
	consulServerList.RUnlock()

	consulServerList.Lock()
	defer consulServerList.Unlock()
	if _, ok := s.serviceWatches[service]; ok {
		return nil
	}

	wp, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": service,
	})
	if err != nil {
		return fmt.Errorf("watch.Parse failed, service: %v, %w", service, err)
	}

	wp.Handler = func(idx uint64, data interface{}) {
		switch dat := data.(type) {
		case []*api.ServiceEntry:
			consulServerList.Lock()
			defer consulServerList.Unlock()

			instances := make([]Server, 0, len(dat))
			for _, entry := range dat {
				if entry.Checks.AggregatedStatus() != ConsulStatusPassing {
					continue
				}
				instances = append(instances, Server{
					Address: entry.Service.Address,
					Port:    entry.Service.Port,
					Meta:    entry.Service.Meta,
				})
			}

			s.servers[service] = instances
			Debugf("Watch receive service changes to %v, %d instances, %d passing instances, instances: %+v",
				service, len(dat), len(instances), instances)
		}
	}

	s.serviceWatches[service] = wp
	go wp.RunWithClientAndHclog(GetConsulClient(), nil)
	rail.Infof("Created Consul Service Watch for %v", service)

	return nil
}

func (s *ServerList) UnsubscribeAll(rail Rail) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.serviceWatches {
		v.Stop()
	}
	rail.Debugf("Stopped all service watches, in total %d watches", len(s.serviceWatches))
	return nil
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
		if name == "consul" || name == consulRegistration.serviceName {
			continue
		}
		err := fetchAndCacheServiceNodes(rail, name)
		if err != nil {
			rail.Warnf("Failed to poll service service for '%s', err: %v", name, err)
		}
	}
}

// Fetch and cache services nodes.
func fetchAndCacheServiceNodes(rail Rail, name string) error {
	consulServerList.Lock()
	defer consulServerList.Unlock()

	services, err := ConsulApi.CatalogFetchServiceNodes(rail, name)
	if err != nil {
		return fmt.Errorf("failed to FetchServicesByName, name: %v, %v", name, err)
	}
	servers := make([]Server, 0, len(services))
	for i := range services {
		s := services[i]
		if s.Checks.AggregatedStatus() != ConsulStatusPassing {
			continue
		}
		servers = append(servers, Server{
			Meta:    s.ServiceMeta,
			Address: s.ServiceAddress,
			Port:    s.ServicePort,
		})
	}
	rail.Debugf("Fetched %d (passing) instances for service: %v, %+v", len(servers), name, servers)
	consulServerList.servers[name] = servers
	return err
}

/*
Resolve request url for the given service.

The resolved url will be in format: "http://" + host + ":" + port + "/" + relUrl.

Return ErrServiceInstanceNotFound if no instance is found.
*/
func ConsulResolveRequestUrl(serviceName string, relUrl string) (string, error) {
	server, err := SelectServer(serviceName, RandomServerSelector)
	if err != nil {
		return "", err
	}
	return server.BuildUrl(relUrl), nil
}

/*
Resolve service address (host:port).

Return ErrServiceInstanceNotFound if no instance is found.
*/
func ConsulResolveServiceAddr(name string) (string, error) {
	selected, err := SelectServer(name, RandomServerSelector)
	if err != nil {
		return "", err
	}
	return selected.ServerAddress(), nil
}

// Select one Server based on the provided selector.
//
// If none is matched, ErrConsulServiceInstanceNotFound is returned.
func SelectServer(name string, selector func(servers []Server) int) (Server, error) {
	consulServerList.RLock()
	defer consulServerList.RUnlock()
	servers := consulServerList.servers[name]

	if len(servers) < 1 {
		return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrConsulServiceInstanceNotFound)
	}
	selected := selector(servers)
	if selected >= 0 && selected < len(servers) {
		return servers[selected], nil
	}
	return Server{}, fmt.Errorf("failed to select server for %v, %w", name, ErrConsulServiceInstanceNotFound)
}

// List Consul Servers already loaded in cache.
func ListServers(name string) []Server {
	consulServerList.RLock()
	defer consulServerList.RUnlock()
	servers := consulServerList.servers[name]
	copied := make([]Server, 0, len(servers))
	copy(copied, servers)
	return copied
}

// Register current service
func DeregisterService() error {
	if !IsConsulClientInitialized() {
		return nil
	}

	consulRegistration.mu.Lock()
	defer consulRegistration.mu.Unlock()

	// not registered
	if consulRegistration.serviceId == ServiceIdNil {
		return nil
	}

	Infof("Deregistering current instance on Consul, service_id: '%s'", consulRegistration.serviceId)

	err := ConsulApi.DeregisterService(consulRegistration.serviceId)
	if err == nil {
		consulRegistration.serviceId = ServiceIdNil
	}
	return err
}

// Check if current instance is registered on consul.
func IsConsulServiceRegistered() bool {
	if !IsConsulClientInitialized() {
		return false
	}

	consulRegistration.mu.Lock()
	defer consulRegistration.mu.Unlock()
	return consulRegistration.serviceId != ServiceIdNil
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
	consulRegistration.mu.Lock()
	defer consulRegistration.mu.Unlock()

	// registered already
	if consulRegistration.serviceId != ServiceIdNil {
		return nil
	}

	serverPort := GetPropInt(PropServerPort)
	registerName := GetPropStr(PropConsuleRegisterName)
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
			Status:                         ConsulStatusPassing, // for responsiveness
		},
		Meta: GetPropStrMap(PropConsulMetadata),
	}

	if err := ConsulApi.RegisterService(registration); err != nil {
		return err
	}
	consulRegistration.serviceId = proposedServiceId
	consulRegistration.serviceName = registerName

	Infof("Registered on Consul, serviceId: '%s'", proposedServiceId)
	return nil
}

// Get the already created consul client.
//
// InitConsulClient() must be called before this func.
//
// If the client is not already created, this func will panic.
func GetConsulClient() *api.Client {
	if !IsConsulClientInitialized() {
		panic("consul client is not initialized")
	}
	return consulp.consul
}

/*
Get or init new consul client

For the first time that the consul client is initialized, this func will look for prop:

	"consul.consulAddress"
*/
func InitConsulClient() error {
	if IsConsulClientInitialized() {
		return nil
	}

	consulp.mu.Lock()
	defer consulp.mu.Unlock()

	if consulp.consul != nil {
		return nil
	}

	addr := GetPropStr(PropConsulAddress)
	c, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		return fmt.Errorf("failed to create new Consul client, %w", err)
	}
	consulp.consul = c
	Infof("Created Consul Client on %s", addr)

	consulServerListPoller = NewTickRuner(
		GetPropDur(PropConsulFetchServerInterval, time.Second),
		func() {
			rail := EmptyRail()
			// make sure we poll service instance right after we created ticker
			PollServiceListInstances(rail)
		})
	consulServerListPoller.Start()

	return nil
}

// Check whether consul client is initialized
func IsConsulClientInitialized() bool {
	consulp.mu.RLock()
	defer consulp.mu.RUnlock()
	return consulp.consul != nil
}

func ConsulBootstrap(rail Rail) error {

	if GetPropBool(PropConsulEnableDeregisterUrl) {
		deregisterUrl := GetPropStr(PropConsulDeregisterUrl)
		if !IsBlankStr(deregisterUrl) {
			rail.Infof("Enabled 'GET %v' for manual consul service deregistration", deregisterUrl)
			Get(deregisterUrl, func(c *gin.Context, rail Rail) (any, error) {
				if !IsConsulServiceRegistered() {
					rail.Info("Current instance is not registered on consul")
					return nil, nil
				}

				rail.Info("deregistering consul service registration")
				if err := DeregisterService(); err != nil {
					rail.Errorf("failed to deregistered consul service, %v", err)
					return nil, err
				} else {
					rail.Info("consul service deregistered")
				}
				return nil, nil
			}).Build()
		}
	}

	// create consul client
	if err := InitConsulClient(); err != nil {
		return fmt.Errorf("failed to create Consul client, %w", err)
	}

	// deregister on shutdown
	AddShutdownHook(func() {

		// deregister current instnace
		if IsConsulServiceRegistered() {
			if e := DeregisterService(); e != nil {
				rail.Errorf("Failed to deregister on Consul, %v", e)
			}
		}

		// stop service instance poller
		if consulServerListPoller != nil {
			consulServerListPoller.Stop()
		}

		// stop service watches
		consulServerList.UnsubscribeAll(rail)
	})

	if e := RegisterService(); e != nil {
		return fmt.Errorf("failed to register on Consul, %w", e)
	}

	ClientServiceRegistry = ConsulBasedServiceRegistry
	rail.Debug("Using ConsulBasedServiceRegistry")

	return nil
}

func ConsulBootstrapCondition(rail Rail) (bool, error) {
	return IsConsulEnabled(), nil
}

// Service registry based on Consul
type ConsulServiceRegistry struct {
	Rule ServerSelector
}

func (c *ConsulServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {

	// select one of the instance for this service
	server, err := SelectServer(service, c.Rule)

	// always try to create a watch for the service
	defer consulServerList.Subscribe(rail, service)

	if err != nil {
		// it's possible that we haven't created a watch for the service
		// and the last time we polled the service instances, there was no instance returned for it
		// we may just try to fetch again and hope for the best
		if errors.Is(err, ErrConsulServiceInstanceNotFound) {

			// already subscribed, give up
			if consulServerList.IsSubscribed(rail, service) {
				return "", err
			}

			// fetch immediately
			if err := fetchAndCacheServiceNodes(rail, service); err != nil {
				return "", err
			}

			// select again
			if server, err = SelectServer(service, c.Rule); err != nil {
				// pretty sure that the service is completely down
				return "", err
			}
		} else {
			// recovered
			return "", err
		}
	}

	return server.BuildUrl(relativeUrl), nil
}

func (c *ConsulServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	return ListServers(service), nil
}

func SubscribeConsulService(rail Rail, service string) error {
	return consulServerList.Subscribe(rail, service)
}
