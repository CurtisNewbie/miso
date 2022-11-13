package gocommon

import (
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

const (
	STATUS_PASSING = "passing"
)

var (
	// Service registration
	regSub = &serviceRegistration{serviceId: SERVICE_ID_NIL}

	// Zero value for empty serviceId
	SERVICE_ID_NIL = "nil"

	// Global handle to the Consul client
	consulp = &consulHolder{consul: nil}

	// Holder (cache) of service list and their instances
	serviceListHolder = &ServiceListHolder{
		Instances:   map[string][]*api.AgentService{},
		ServiceList: Set[string]{},
	}

	// server list polling subscription
	serverListPSub = &serverListPollingSubscription{sub: nil}
)

type serverListPollingSubscription struct {
	sub *time.Timer
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
	mu          sync.Mutex
	Instances   map[string][]*api.AgentService
	ServiceList Set[string]
}

func init() {
	SetDefProp(PROP_CONSUL_ENABLED, false)
	SetDefProp(PROP_CONSUL_CONSUL_ADDRESS, "localhost:8500")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_URL, "/health")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_INTERVAL, "60s")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_TIMEOUT, "3s")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER, "130s")
}

// Subscribe to server list
func SubscribeServerList() {
	serverListPSub.mu.Lock()
	defer serverListPSub.mu.Unlock()

	if serverListPSub.sub != nil {
		return
	}
}

// Unsubscribe to server list
func UnsubscribeServerList() {
	serverListPSub.mu.Lock()
	defer serverListPSub.mu.Unlock()

	if serverListPSub.sub == nil {
		return
	} else {
		serverListPSub.sub.Stop()
	}

	serverListPSub.sub = time.NewTimer(30 * time.Second)
	go func() {
		PollServiceListInstances()
		<-serverListPSub.sub.C
	}()
}

/*
	Check if consul is enabled

	This func looks for following prop:

		PROP_CONSUL_ENABLED
*/
func IsConsulEnabled() bool {
	return GetPropBool(PROP_CONSUL_ENABLED)
}

// Poll all service list and cache them
func PollServiceListInstances() {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

	logrus.Info("Polling service list")
	for k := range serviceListHolder.ServiceList {
		_, err := _fetchAndCacheServicesByName(k)
		if err != nil {
			logrus.Warnf("Failed to poll service service for '%s', err: %v", k, err)
		}
	}
}

// Fetch services by name and cache the result from Consul, this func requires extra lock
func _fetchAndCacheServicesByName(name string) (map[string]*api.AgentService, error) {
	services, err := FetchServicesByName(name)
	if err != nil {
		return nil, err
	}
	serviceListHolder.Instances[name] = ValuesOfStMap(services)
	return services, err
}

/*
	Resolve service address (host:port)

	This func will first read the cache, trying to resolve the services address
	without actually requesting consul, and only when the cache missed, it then
	requests the consul
*/
func ResolveServiceAddress(name string) (string, error) {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

	serviceListHolder.ServiceList[name] = Void{}
	instances := serviceListHolder.Instances[name]
	if instances == nil {
		_fetchAndCacheServicesByName(name)
		instances = serviceListHolder.Instances[name]
	}

	// no instances available
	if instances == nil || len(instances) < 1 {
		return "", fmt.Errorf("unable to find any available service instance for '%s'", name)
	}
	return extractServiceAddress(RandomOne(instances)), nil
}

// Create a default health check endpoint that simply doesn't nothing except returing 200
func DefaultHealthCheck(ctx *gin.Context) {
	ctx.Status(200)
}

// Extract service address (host:port) from Agent.Service
func extractServiceAddress(agent *api.AgentService) string {
	if agent != nil {
		return fmt.Sprintf("%s:%d", agent.Address, agent.Port)
	}
	return ""
}

// Fetch service address (host:port, without protocol)
func FetchServiceAddress(name string) (string, error) {
	services, err := FetchServicesByName(name)
	if err != nil {
		return "", err
	}
	agent := RandomOne(ValuesOfStMap(services))
	return extractServiceAddress(agent), nil
}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func FetchServicesByName(name string) (map[string]*api.AgentService, error) {
	client, err := GetConsulClient()
	if err != nil {
		return nil, err
	}

	logrus.Infof("Requesting services for '%s' from Consul", name)
	services, err := client.Agent().ServicesWithFilter(fmt.Sprintf("Service == \"%s\"", name))
	if err != nil {
		panic(err)
	}
	return services, nil
}

// Fetch all registered services, this method always call Consul instead of reading from cache
func FetchServices() (map[string]*api.AgentService, error) {
	client, e := GetConsulClient()
	if e != nil {
		return nil, e
	}

	return client.Agent().Services()
}

// Register current service 
func DeregisterService() error {
	if !IsConsulClientInitialized() {
		return nil
	}

	regSub.mu.Lock()
	defer regSub.mu.Unlock()

	// not registered
	if regSub.serviceId == SERVICE_ID_NIL {
		return nil
	}

	logrus.Infof("Deregistering current instance on Consul, service_id: %s", regSub.serviceId)
	client, _ := GetConsulClient()
	err := client.Agent().ServiceDeregister(regSub.serviceId)

	// zero the serviceId
	if err != nil {
		regSub.serviceId = SERVICE_ID_NIL
	}

	return err
}

/*
	Register current instance as a service

	If we have already registered before, current method call will be ignored.

	This func looks for following prop:

		PROP_SERVER_PORT
		PROP_CONSUL_REGISTER_NAME
		PROP_CONSUL_HEALTHCHECK_INTERVAL
		PROP_CONSUL_REGISTER_ADDRESS
		PROP_CONSUL_HEALTHCHECK_URL
		PROP_CONSUL_HEALTHCHECK_TIMEOUT
		PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER
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
	if regSub.serviceId != SERVICE_ID_NIL {
		return nil
	}

	serverPort := GetPropInt(PROP_SERVER_PORT)
	registerName := GetPropStr(PROP_CONSUL_REGISTER_NAME)
	registerAddress := GetPropStr(PROP_CONSUL_REGISTER_ADDRESS)
	healthCheckUrl := GetPropStr(PROP_CONSUL_HEALTHCHECK_URL)
	healthCheckInterval := GetPropStr(PROP_CONSUL_HEALTHCHECK_INTERVAL)
	healthCheckTimeout := GetPropStr(PROP_CONSUL_HEALTHCHECK_TIMEOUT)
	healthCheckDeregAfter := GetPropStr(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = ResolveServerHost(GetPropStr(PROP_SERVER_HOST))
	}

	proposedServiceId := fmt.Sprintf("%s:%d:%s", registerName, serverPort, RandStr(5))
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
			// Status:                         STATUS_PASSING,
		},
	}
	logrus.Infof("Registering current instance as a service on Consul, registration: %+v, check: %+v", registration, registration.Check)

	if e = client.Agent().ServiceRegister(registration); e != nil {
		logrus.Errorf("Failed to register on Consul, err: %v", e)
		return e
	}
	regSub.serviceId = proposedServiceId

	return nil
}

/*
	Get or init new consul client

	For the first time that the consul client is initialized, this func will look for prop:

		PROP_CONSUL_CONSUL_ADDRESS
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

	c, err := api.NewClient(&api.Config{
		Address: GetPropStr(PROP_CONSUL_CONSUL_ADDRESS),
	})
	if err != nil {
		return nil, err
	}
	consulp.consul = c

	SubscribeServerList()

	return c, nil
}

// Check whether consul client is initialized
func IsConsulClientInitialized() bool {
	consulp.mu.RLock()
	defer consulp.mu.RUnlock()
	return consulp.consul != nil
}
