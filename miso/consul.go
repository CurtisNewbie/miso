package miso

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
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
		ServiceList: NewSet[string](),
	}

	// server list polling subscription
	serverListPSub = &serverListPollingSubscription{sub: nil}

	ErrServiceInstanceNotFound = errors.New("unable to find any available service instance")
)

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
	mu          sync.Mutex
	Instances   map[string][]*api.AgentService
	ServiceList Set[string]
}

func init() {
	SetDefProp(PROP_CONSUL_ENABLED, false)
	SetDefProp(PROP_CONSUL_CONSUL_ADDRESS, "localhost:8500")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_URL, "/health")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_INTERVAL, "15s")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_TIMEOUT, "3s")
	SetDefProp(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER, "120s")
	SetDefProp(PROP_CONSUL_REGISTER_DEFAULT_HEALTHCHECK, true)
}

// Subscribe to server list, refresh server list every 30s
func SubscribeServerList() {
	DoSubscribeServerList(30)
}

// Subscribe to server list
func DoSubscribeServerList(everyNSec int) {
	serverListPSub.mu.Lock()
	defer serverListPSub.mu.Unlock()

	if serverListPSub.sub != nil {
		return
	}

	serverListPSub.sub = time.NewTicker(time.Duration(everyNSec) * time.Second)
	c := serverListPSub.sub.C
	go func() {
		for {
			PollServiceListInstances()
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
	return GetPropBool(PROP_CONSUL_ENABLED)
}

// Poll all service list and cache them
func PollServiceListInstances() {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

	for k := range serviceListHolder.ServiceList.Keys {
		_, err := _fetchAndCacheServicesByName(k)
		if err != nil {
			EmptyRail().Warnf("Failed to poll service service for '%s', err: %v", k, err)
		}
	}
}

// Fetch services by name and cache the result from Consul, this func requires extra lock
func _fetchAndCacheServicesByName(name string) (map[string]*api.AgentService, error) {
	services, err := FetchServicesByName(name)
	if err != nil {
		return nil, err
	}
	serviceListHolder.Instances[name] = ValuesOfMap(&services)
	return services, err
}

/*
Resolve request url for the given service.

	"http://" + host ":" + port + "/" + relUrl

This func will first read the cache, trying to resolve the services address
without actually requesting consul, and only when the cache missed, it then
requests the consul.

Return consul.ErrServiceInstanceNotFound if no instance available
*/
func ResolveRequestUrl(serviceName string, relUrl string) (string, error) {
	if !strings.HasPrefix(relUrl, "/") {
		relUrl = "/" + relUrl
	}

	address, err := ResolveServiceAddress(serviceName)
	if err != nil {
		return "", err
	}

	return "http://" + address + relUrl, nil
}

/*
Resolve service address (host:port)

This func will first read the cache, trying to resolve the services address
without actually requesting consul, and only when the cache missed, it then
requests the consul

Return consul.ErrServiceInstanceNotFound if no instance available
*/
func ResolveServiceAddress(name string) (string, error) {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

	serviceListHolder.ServiceList.Add(name)
	instances := serviceListHolder.Instances[name]
	if instances == nil {
		_fetchAndCacheServicesByName(name)
		instances = serviceListHolder.Instances[name]
	}

	// no instances available
	if instances == nil || len(instances) < 1 {
		return "", ErrServiceInstanceNotFound
	}
	return extractServiceAddress(RandomOne(instances)), nil
}

// Create a default health check endpoint that simply doesn't nothing except returing 200
func DefaultHealthCheck(ctx *gin.Context) {
	ctx.String(http.StatusOK, "UP")
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
	agent := RandomOne(ValuesOfMap(&services))
	return extractServiceAddress(agent), nil
}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func FetchServicesByName(name string) (map[string]*api.AgentService, error) {
	client, err := GetConsulClient()
	if err != nil {
		return nil, err
	}

	services, err := client.Agent().ServicesWithFilter(fmt.Sprintf("Service == \"%s\"", name))
	if err != nil {
		return nil, err
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

	EmptyRail().Infof("Deregistering current instance on Consul, service_id: '%s'", regSub.serviceId)
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
	if regSub.serviceId != SERVICE_ID_NIL {
		return nil
	}

	serverPort := GetPropInt(PROP_SERVER_PORT)
	registerName := GetPropStr(PROP_CONSUL_REGISTER_NAME)
	if registerName == "" { // fallback to app.name
		registerName = GetPropStr(PROP_APP_NAME)
	}
	registerAddress := GetPropStr(PROP_CONSUL_REGISTER_ADDRESS)
	healthCheckUrl := GetPropStr(PROP_CONSUL_HEALTHCHECK_URL)
	healthCheckInterval := GetPropStr(PROP_CONSUL_HEALTHCHECK_INTERVAL)
	healthCheckTimeout := GetPropStr(PROP_CONSUL_HEALTHCHECK_TIMEOUT)
	healthCheckDeregAfter := GetPropStr(PROP_CONSUL_HEALTHCHECK_FAILED_DEREG_AFTER)

	// registerAddress not specified, resolve the ip address used for the server
	if registerAddress == "" {
		registerAddress = ResolveServerHost(GetPropStr(PROP_SERVER_HOST))
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
			Status:                         STATUS_PASSING, // for responsiveness
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

	addr := GetPropStr(PROP_CONSUL_CONSUL_ADDRESS)
	c, err := api.NewClient(&api.Config{
		Address: addr,
	})
	if err != nil {
		return nil, err
	}
	consulp.consul = c
	EmptyRail().Infof("Created Consul Client on %s", addr)

	SubscribeServerList()

	return c, nil
}

// Check whether consul client is initialized
func IsConsulClientInitialized() bool {
	consulp.mu.RLock()
	defer consulp.mu.RUnlock()
	return consulp.consul != nil
}
