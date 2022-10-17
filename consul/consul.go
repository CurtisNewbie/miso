package consul

import (
	"fmt"
	"strconv"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

var (
	// Global handle to the Consul client
	consulClient *api.Client
	serviceId    *string
)

// Register a default health check endpoint ('/health') on GIN
func RegisterDefaultHealthCheck(engine *gin.Engine) {
	engine.GET("/health", DefaultHealthCheck)
}

// Create a default health check endpoint that simply doesn't nothing except returing 200
func DefaultHealthCheck(ctx *gin.Context) {
	ctx.Status(200)
}

// Fetch service address (host:port), this method always call Consul instead of reading from cache
func FetchServiceAddress(name string) (string, error) {
	service, err := FetchService(name)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", service.Address, service.Port), nil
}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func FetchService(name string) (*api.AgentService, error) {
	services, err := FetchServices()
	if err != nil {
		return nil, err
	}
	return services[name], nil
}

// Fetch all registered services, this method always call Consul instead of reading from cache
func FetchServices() (map[string]*api.AgentService, error) {
	return GetConsulClient().Agent().Services()
}

// Register current service
func DeregisterService(consulConf *config.ConsulConfig) {
	if serviceId == nil {
		return
	}

	GetConsulClient().Agent().ServiceDeregister(*serviceId)
}

// Register current instance as a service
func RegisterService(consulConf *config.ConsulConfig, serverConf *config.ServerConfig) error {
	i_port, _ := strconv.Atoi(serverConf.Port)
	*serviceId = fmt.Sprintf("%s:%s:%s", consulConf.RegisterName, serverConf.Port, util.RandStr(5))

	registration := &api.AgentServiceRegistration{
		ID:      *serviceId,
		Name:    consulConf.RegisterName,
		Port:    i_port,
		Address: serverConf.Host,
		Check: &api.AgentServiceCheck{
			HTTP:     consulConf.HealthCheckUrl,
			Interval: consulConf.HealthCheckInterval,
			Timeout:  consulConf.HealthCheckTimeout,
		},
	}
	logrus.Infof("Registering current instance as a service to Consul, service_id: %s, service_name: %s", *serviceId, consulConf.RegisterName)
	return GetConsulClient().Agent().ServiceRegister(registration)
}

// Get the initialized Consul client, InitConsulClient should be called first before this method
func GetConsulClient() *api.Client {
	if consulClient == nil {
		panic("Consul Client is not initialized")
	}
	return consulClient
}

// Init new Consul Client
func InitConsulClient(consulConf *config.ConsulConfig) (*api.Client, error) {
	if consulClient != nil {
		return consulClient, nil
	}

	c, err := api.NewClient(&api.Config{
		Address: consulConf.ConsulAddress,
	})
	if err != nil {
		return nil, err
	}

	consulClient = c
	return c, nil
}
