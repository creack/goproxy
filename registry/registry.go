// Package registry defines the Registry interface which can be used with goproxy.
package registry

import (
	"errors"
	"log"
	"sync"
)

// Global lock for the default registry.
var lock sync.RWMutex

// Common errors.
var (
	ErrServiceNotFound = errors.New("service name/version not found")
)

// Registry is an interface used to lookup the target host
// for a given service name / version pair.
type Registry interface {
	Add(name, version, endpoint string)                // Add an endpoint to our registry.
	DeleteService(name string)                         // Remove an endpoint to our registry.
	DeleteVersion(name, version string)                // Remove an endpoint to our registry.
	DeleteEndpoint(name, version, endpoint string)     // Remove an endpoint to our registry.
	Failure(name, version, endpoint string, err error) // Mark an endpoint as failed.
	Lookup(name, version string) ([]string, error)     // Return the endpoint list for the given service name/version.
}

// DefaultRegistry is a basic registry using the following format:
// {
//   "serviceName": {
//     "serviceVersion": [
//       "endpoint1:port",
//       "endpoint2:port"
//     ],
//   },
// }
type DefaultRegistry map[string]map[string][]string

// Lookup return the endpoint list for the given service name/version.
func (reg DefaultRegistry) Lookup(name, version string) ([]string, error) {
	lock.RLock()
	targets, ok := reg[name][version]
	lock.RUnlock()
	if !ok {
		return nil, ErrServiceNotFound
	}
	return targets, nil
}

// Failure marks the given endpoint for service name/version as failed.
func (reg DefaultRegistry) Failure(name, version, endpoint string, err error) {
	// Would be used to remove an endpoint from the rotation, log the failure, etc.
	log.Printf("Error accessing %s/%s (%s): %s", name, version, endpoint, err)
}

// Add adds the given endpoit for the service name/version.
func (reg DefaultRegistry) Add(name, version, endpoint string) {
	lock.Lock()
	defer lock.Unlock()

	service, ok := reg[name]
	if !ok {
		service = map[string][]string{}
		reg[name] = service
	}
	service[version] = append(service[version], endpoint)
}

// DeleteEndpoint removes the given endpoit for the service name/version.
func (reg DefaultRegistry) DeleteEndpoint(name, version, endpoint string) {
	lock.Lock()
	defer lock.Unlock()

	service, ok := reg[name]
	if !ok {
		return
	}
begin:
	for i, svc := range service[version] {
		if svc == endpoint {
			copy(service[version][i:], service[version][i+1:])
			service[version][len(service[version])-1] = ""
			service[version] = service[version][:len(service[version])-1]
			goto begin
		}
	}
}

// DeleteVersion removes the given version for the service name.
func (reg DefaultRegistry) DeleteVersion(name, version string) {
	lock.Lock()
	defer lock.Unlock()

	service, ok := reg[name]
	if !ok {
		return
	}
	delete(service, version)
}

// DeleteService removes the given service.
func (reg DefaultRegistry) DeleteService(name string) {
	lock.Lock()
	defer lock.Unlock()

	delete(reg, name)
}
