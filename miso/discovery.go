package miso

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

var (
	// Select Server randomly.
	RandomServerSelector ServerSelector = func(servers []Server) int {
		return rand.Int() % len(servers)
	}
	PropBasedServiceRegistry                 = &HardcodedServiceRegistry{}
	ClientServiceRegistry    ServiceRegistry = nil

	ErrMissingServiceName            = errors.New("service name is required")
	ErrConsulServiceInstanceNotFound = errors.New("unable to find any available service instance")
)

// Server selector, returns index of the selected one.
type ServerSelector func(servers []Server) int

// Server.
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

// Build server address with host and port concatenated.
func (c *Server) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Address, c.Port)
}

// Service registry
type ServiceRegistry interface {

	// Resolve request url.
	ResolveUrl(rail Rail, service string, relativeUrl string) (string, error)

	// List all instances of the service.
	ListServers(rail Rail, service string) ([]Server, error)
}

// Get service registry.
//
// Service registry initialization is lazy, don't store the retunred value in global var.
func GetServiceRegistry() ServiceRegistry {
	if ClientServiceRegistry != nil {
		return ClientServiceRegistry
	}
	return PropBasedServiceRegistry
}

// Service registry backed by loaded configuration.
type HardcodedServiceRegistry struct {
}

func (r *HardcodedServiceRegistry) ResolveUrl(rail Rail, service string, relativeUrl string) (string, error) {
	if IsBlankStr(service) {
		return "", ErrMissingServiceName
	}

	host := serverHostFromProp(service)
	port := serverPortFromProp(service)

	if IsBlankStr(host) {
		return httpProto + service + relativeUrl, nil
	}

	return httpProto + fmt.Sprintf("%v:%v", host, port) + relativeUrl, nil
}

func (r *HardcodedServiceRegistry) ListServers(rail Rail, service string) ([]Server, error) {
	if IsBlankStr(service) {
		return []Server{}, ErrMissingServiceName
	}

	host := serverHostFromProp(service)
	port := serverPortFromProp(service)

	if IsBlankStr(host) {
		return []Server{{Address: host, Port: port, Meta: map[string]string{}}}, nil
	}

	return []Server{{Address: service, Port: 0, Meta: map[string]string{}}}, nil
}

func serverHostFromProp(name string) string {
	if name == "" {
		return ""
	}
	return GetPropStr("client.addr." + name + ".host")
}

func serverPortFromProp(name string) int {
	if name == "" {
		return 0
	}
	return GetPropInt("client.addr." + name + ".port")
}
