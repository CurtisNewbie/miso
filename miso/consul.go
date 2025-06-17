//go:build !excl_consul
// +build !excl_consul

package miso

import (
	"fmt"
	"sync"
	"time"

	"github.com/curtisnewbie/miso/util"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/spf13/cast"
)

const (
	// Service registration status - passing.
	ConsulStatusPassing = "passing"
)

var (
	_ ServerList = (*ConsulServerList)(nil)
)

var (
	// Service registration
	consulRegistration = &serviceRegistration{serviceId: ServiceIdNil}

	// Global handle to the Consul client
	consulp = &consulHolder{consul: nil}

	// Holder (cache) of service list and their instances
	consulServerList = &ConsulServerList{
		servers:        map[string][]Server{},
		serviceWatches: map[string]*watch.Plan{},
	}

	// server list polling subscription
	consulServerListPoller *TickRunner = nil
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
	RegisterBootstrapCallback(ComponentBootstrap{
		Name:      "Boostrap Consul",
		Bootstrap: consulBootstrap,
		Condition: consulBootstrapCondition,
		Order:     BootstrapOrderL4,
	})
}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func CatalogFetchServiceNodes(rail Rail, name string) ([]*api.CatalogService, error) {
	defer DebugTimeOp(rail, time.Now(), "CatalogFetchServiceNodes")
	client := GetConsulClient()

	services, _, err := client.Catalog().Service(name, "", nil)
	if err != nil {
		return nil, err
	}
	return services, nil
}

// Fetch all registered services, this method always call Consul instead of reading from cache
func CatalogFetchServiceNames(rail Rail) (map[string][]string, error) {
	client := GetConsulClient()
	services, _, err := client.Catalog().Services(nil)
	rail.Debugf("CatalogFetchServiceNames, %+v, %v", services, err)
	return services, err
}

// Holder of a list of ServiceHolder
type ConsulServerList struct {
	sync.RWMutex
	servers        map[string][]Server
	serviceWatches map[string]*watch.Plan
}

func (s *ConsulServerList) PollInstances(rail Rail) error {
	names, err := CatalogFetchServiceNames(rail)
	if err != nil {
		return fmt.Errorf("failed to CatalogFetchServiceNames, %w", err)
	}

	for name := range names {
		if name == "consul" {
			continue
		}
		err := s.PollInstance(rail, name)
		if err != nil {
			return fmt.Errorf("failed to poll service service for '%s', err: %w", name, err)
		}
	}
	return nil
}

func (s *ConsulServerList) ListServers(rail Rail, name string) []Server {
	s.RLock()
	defer s.RUnlock()
	servers := s.servers[name]
	copied := make([]Server, len(servers))
	copy(copied, servers)
	return copied
}

func (s *ConsulServerList) IsSubscribed(rail Rail, service string) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.serviceWatches[service]
	return ok
}

func (s *ConsulServerList) Subscribe(rail Rail, service string) error {
	s.RLock()
	if _, ok := s.serviceWatches[service]; ok {
		s.RUnlock()
		return nil
	}
	s.RUnlock()

	s.Lock()
	defer s.Unlock()
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
			s.Lock()
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
			s.Unlock()

			TriggerServerChangeListeners(service)
		}
	}

	s.serviceWatches[service] = wp
	go wp.RunWithClientAndHclog(GetConsulClient(), nil)
	Infof("Created Consul Service Watch for %v", service)

	return nil
}

func (s *ConsulServerList) UnsubscribeAll(rail Rail) error {
	s.Lock()
	defer s.Unlock()
	for _, v := range s.serviceWatches {
		v.Stop()
	}
	rail.Debugf("Stopped all service watches, in total %d watches", len(s.serviceWatches))
	return nil
}

func (s *ConsulServerList) Unsubscribe(rail Rail, service string) error {
	s.Lock()
	defer s.Unlock()
	if v, ok := s.serviceWatches[service]; ok {
		v.Stop()
		rail.Debugf("Stopped service watch for %v", service)
	} else {
		rail.Debugf("Service watch for %v is not found", service)
	}
	return nil
}

// Fetch and cache services nodes.
func (s *ConsulServerList) PollInstance(rail Rail, name string) error {
	s.Lock()
	defer s.Unlock()

	services, err := CatalogFetchServiceNodes(rail, name)
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
	s.servers[name] = servers
	return err
}

// Deregister current service
func DeregisterConsulService() error {
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

	err := GetConsulClient().Agent().ServiceDeregister(consulRegistration.serviceId)
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
func RegisterConsulService() error {
	consulRegistration.mu.Lock()
	defer consulRegistration.mu.Unlock()

	// registered already
	if consulRegistration.serviceId != ServiceIdNil {
		return nil
	}

	serverPort := GetPropInt(PropServerActualPort)
	registerName := GetPropStr(PropConsuleRegisterName)
	registerAddress := GetPropStr(PropConsulRegisterAddress)
	healthCheckUrl := healthCheckUrl()
	healthCheckInterval := GetPropStr(PropHealthCheckInterval)
	healthCheckTimeout := GetPropStr(PropHealthcheckTimeout)
	healthCheckDeregAfter := GetPropStr(PropConsulHealthCheckFailedDeregAfter)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = ResolveServerHost(GetPropStr(PropServerHost))
	} else {
		registerAddress = ResolveServerHost(registerAddress)
	}

	meta := GetPropStrMap(PropConsulMetadata)
	if meta == nil {
		meta = map[string]string{}
	}
	meta[ServiceMetaRegisterTime] = cast.ToString(util.Now().UnixMilli())

	completeHealthCheckUrl := fmt.Sprintf("http://%s:%v%s", registerAddress, serverPort, healthCheckUrl)
	proposedServiceId := fmt.Sprintf("%s-%d", registerName, serverPort)
	registration := &api.AgentServiceRegistration{
		ID:      proposedServiceId,
		Name:    registerName,
		Port:    serverPort,
		Address: registerAddress,
		Check: &api.AgentServiceCheck{
			HTTP:                           completeHealthCheckUrl,
			Interval:                       healthCheckInterval,
			Timeout:                        healthCheckTimeout,
			DeregisterCriticalServiceAfter: healthCheckDeregAfter,
			Status:                         ConsulStatusPassing, // for responsiveness (TODO)
		},
		Meta: meta,
	}

	if err := GetConsulClient().Agent().ServiceRegister(registration); err != nil {
		return WrapErrf(err, "failed to register consul service")
	}
	consulRegistration.serviceId = proposedServiceId
	consulRegistration.serviceName = registerName

	Infof("Registered on Consul, serviceId: '%s', healthCheckUrl: '%v'", proposedServiceId, completeHealthCheckUrl)
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
			if err := consulServerList.PollInstances(rail); err != nil {
				rail.Errorf("failed to poll consul service instances, %v", err)
			}
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

func consulBootstrap(rail Rail) error {

	// setup api
	ChangeGetServerList(func() ServerList { return consulServerList })
	rail.Debug("Using Consul based GetServerList")

	if GetPropBool(PropConsulEnableDeregisterUrl) {
		deregisterUrl := GetPropStr(PropConsulDeregisterUrl)
		if !util.IsBlankStr(deregisterUrl) {
			rail.Infof("Enabled 'GET %v' for manual consul service deregistration", deregisterUrl)

			HttpGet(deregisterUrl, ResHandler(
				func(inb *Inbound) (any, error) {
					if !IsConsulServiceRegistered() {
						rail.Info("Current instance is not registered on consul")
						return nil, nil
					}

					rail.Info("deregistering consul service registration")
					if err := DeregisterConsulService(); err != nil {
						rail.Errorf("failed to deregistered consul service, %v", err)
						return nil, err
					} else {
						rail.Info("consul service deregistered")
					}
					return nil, nil
				})).
				Desc("Endpoint used to trigger Consul service deregistration")
		}
	}

	// create consul client
	if err := InitConsulClient(); err != nil {
		return fmt.Errorf("failed to create Consul client, %w", err)
	}

	// deregister on shutdown, we specify the order explicitly to make sure the service
	// is deregistered before shutting down the web server
	AddOrderedShutdownHook(DefShutdownOrder-1, func() {
		rail := EmptyRail()

		if IsConsulServiceRegistered() {
			if e := DeregisterConsulService(); e != nil {
				rail.Errorf("Failed to deregister on Consul, %v", e)
			}
		}

		// stop service instance poller
		if consulServerListPoller != nil {
			consulServerListPoller.Stop()
		}

		// stop service watches
		if err := consulServerList.UnsubscribeAll(rail); err != nil {
			rail.Warnf("failed to UnsubscribeAll all, %v", err)
		}
	})

	if e := RegisterConsulService(); e != nil {
		return fmt.Errorf("failed to register on Consul, %w", e)
	}

	return nil
}

func consulBootstrapCondition(rail Rail) (bool, error) {
	return GetPropBool(PropConsulEnabled), nil
}
