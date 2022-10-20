package consul

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

var (
	// Consul client is not initialized
	errClientNotInit = errors.New("consul client is not initialized")

	// Global handle to the Consul client
	consulClient *api.Client
	// ServiceId used for service registration on Consul
	serviceId *string

	// Holder (cache) of service list and their instances
	serviceListHolder = &ServiceListHolder{
		Instances:   map[string][]*api.AgentService{},
		ServiceList: util.Set[string]{},
	}
)

// Holder of a list of ServiceHolder
type ServiceListHolder struct {
	mu          sync.Mutex
	Instances   map[string][]*api.AgentService
	ServiceList util.Set[string]
}

// Poll all service list and cache them
func PollServiceListInstances() {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

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
	serviceListHolder.Instances[name] = util.ValuesOfStMap(services)
	return services, err
}

// Resolve service address (host:port)
//
// This func will first read the cache, trying to resolve the services address
// without actually requesting consul, and only when the cache missed, it then
// requests the consul
func ResolveServiceAddress(name string) (string, error) {
	serviceListHolder.mu.Lock()
	defer serviceListHolder.mu.Unlock()

	serviceListHolder.ServiceList[name] = util.Void{}
	instances := serviceListHolder.Instances[name]
	if instances == nil {
		_fetchAndCacheServicesByName(name)
		instances = serviceListHolder.Instances[name]
	}

	// no instances available
	if instances == nil || len(instances) < 1 {
		return "", fmt.Errorf("unable to find any available service instance for '%s'", name)
	}
	return extractServiceAddress(util.RandomOne(instances)), nil
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
	agent := util.RandomOne(util.ValuesOfStMap(services))
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
func DeregisterService(consulConf *config.ConsulConfig) {
	if consulConf == nil || serviceId == nil {
		return
	}

	logrus.Infof("Deregistering current instance to Consul, service_id: %s", *serviceId)
	client, e := GetConsulClient()
	if e != nil {
		return
	}

	client.Agent().ServiceDeregister(*serviceId)
}

// Register current instance as a service
func RegisterService(consulConf *config.ConsulConfig, serverConf *config.ServerConfig) error {
	util.NonNil(consulConf, "consulConf is nil")
	util.NonNil(serverConf, "serverConf is nil")

	client, e := GetConsulClient()
	if e != nil {
		return e
	}

	i_port, _ := strconv.Atoi(serverConf.Port)
	si := fmt.Sprintf("%s:%s:%s", consulConf.RegisterName, serverConf.Port, util.RandStr(5))
	serviceId = &si

	registration := &api.AgentServiceRegistration{
		ID:      *serviceId,
		Name:    consulConf.RegisterName,
		Port:    i_port,
		Address: consulConf.RegisterAddress,
		Check: &api.AgentServiceCheck{
			HTTP:     "http://" + serverConf.Host + ":" + serverConf.Port + consulConf.HealthCheckUrl,
			Interval: consulConf.HealthCheckInterval,
			Timeout:  consulConf.HealthCheckTimeout,
		},
	}
	logrus.Infof("Registering current instance as a service on Consul, registration: %+v, check: %+v", registration, registration.Check)

	return client.Agent().ServiceRegister(registration)
}

// Get the initialized Consul client, InitConsulClient should be called first before this method
func GetConsulClient() (*api.Client, error) {
	if consulClient == nil {
		return nil, errClientNotInit
	}
	return consulClient, nil
}

// Check whether we have Consul client initialized
func HasConsulClient() bool {
	return consulClient != nil
}

// Init new Consul Client
func InitConsulClient(consulConf *config.ConsulConfig) (*api.Client, error) {
	if consulClient != nil {
		return consulClient, nil
	}

	util.NonNil(consulConf, "consulConf is nil")

	c, err := api.NewClient(&api.Config{
		Address: consulConf.ConsulAddress,
	})
	if err != nil {
		return nil, err
	}

	consulClient = c
	return c, nil
}
