package consul

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/curtisnewbie/gocommon/config"
	"github.com/curtisnewbie/gocommon/util"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
)

var (
	errClientNotInit = errors.New("consul client is not initialized")
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

// Fetch service address (host:port, without protocol), this method always call Consul instead of reading from cache
func FetchServiceAddress(name string) (string, error) {
	services, err := FetchServicesByName(name)
	if err != nil {
		return "", err
	}
	agent := util.RandomOne(util.ValuesOfStMap(services))
	if agent != nil {
		return fmt.Sprintf("%s:%d", agent.Address, agent.Port), nil
	}
	return "", nil
}

// Fetch registered service by name, this method always call Consul instead of reading from cache
func FetchServicesByName(name string) (map[string]*api.AgentService, error) {
	client, err := GetConsulClient()
	if err != nil {
		return nil, err
	}

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

	ipv4 := util.GetLocalIPV4()

	// only use serverConf.Host when it's localhost
	address := serverConf.Host
	if strings.ToLower(address) != "localhost" {
		address = ipv4
	}

	healthCheckUrl := consulConf.HealthCheckUrl
	if healthCheckUrl == "" {
		// default health endpoint (/health)
		healthCheckUrl = "http://" + address + ":" + serverConf.Port + "/health"
		logrus.Infof("Using default health check endpoint: '%s'", healthCheckUrl)
	}

	registration := &api.AgentServiceRegistration{
		ID:      *serviceId,
		Name:    consulConf.RegisterName,
		Port:    i_port,
		Address: address,
		Check: &api.AgentServiceCheck{
			HTTP:     healthCheckUrl,
			Interval: consulConf.HealthCheckInterval,
			Timeout:  consulConf.HealthCheckTimeout,
		},
	}
	logrus.Infof("Registering current instance as a service to Consul, registration: %+v", registration)

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
