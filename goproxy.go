// Package goproxy is a LoadBalancer based on httputil.ReverseProxy.
//
// ExtractNameVersion and LoadBalance can be overridden in order to customize
// the behavior.
package goproxy

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/creack/goproxy/registry"
)

// Common errors
var (
	ErrInvalidService = errors.New("invalid service/version")
)

// ExtractNameVersion is called to lookup the service name / version from
// the requested URL. It should update the URL's Path to reflect the target
// expectation.
var ExtractNameVersion = extractNameVersion

// LoadBalance is the default balancer which will use a random endpoint
// for the given service name/version.
var LoadBalance = loadBalance

// extractNameVersion lookup the target path and extract the name and version.
// It updates the target Path trimming version and name.
// Expected format: `/<name>/<version>/...`
func extractNameVersion(target *url.URL) (name, version string, err error) {
	path := target.Path
	if len(path) > 1 && path[0] == '/' {
		path = path[1:]
	}
	tmp := strings.Split(path, "/")
	if len(tmp) < 2 {
		return "", "", fmt.Errorf("Invalid path")
	}
	name, version = tmp[0], tmp[1]
	target.Path = "/" + strings.Join(tmp[2:], "/")
	return name, version, nil
}

var dialer = (&net.Dialer{
	Timeout:   2 * time.Second,
	KeepAlive: 10 * time.Second,
}).Dial

// loadBalance is a basic loadBalancer which randomly
// tries to connect to one of the endpoints and try again
// in case of failure.
func loadBalance(network, serviceName, serviceVersion string, reg registry.Registry) (net.Conn, error) {
	endpoints, err := reg.Lookup(serviceName, serviceVersion)
	if err != nil {
		return nil, err
	}
	for {
		// No more endpoint, stop
		if len(endpoints) == 0 {
			break
		}
		// Select a random endpoint
		i := rand.Int() % len(endpoints)
		endpoint := endpoints[i]

		// Try to connect
		conn, err := dialer(network, endpoint)
		if err != nil {
			reg.Failure(serviceName, serviceVersion, endpoint, err)
			// Failure: remove the endpoint from the current list and try again.
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		// Success: return the connection.
		return conn, nil
	}
	// No available endpoint.
	return nil, fmt.Errorf("No endpoint available for %s/%s", serviceName, serviceVersion)
}

// IsWebsocket checks if the given request is a websocket.
func IsWebsocket(req *http.Request) (b bool) {
	if c := req.Header.Get("Connection"); c == "" || strings.ToLower(c) != "upgrade" {
		return false
	}
	if u := req.Header.Get("Upgrade"); u == "" || strings.ToLower(u) != "websocket" {
		return false
	}
	return true
}

// NewMultipleHostReverseProxy creates a reverse proxy handler
// that will load balance using the given registry.
// Optionnaly, a logger can be set to handle error outputs and
// a middleware can be given.
// The middleware receive the name and version as well as the handler. Useful for logging/metrics.
func NewMultipleHostReverseProxy(reg registry.Registry, errorLog *log.Logger, middleware func(name, version string, handler http.Handler) http.Handler) http.HandlerFunc {
	transport := &http.Transport{
		MaxIdleConnsPerHost:   50,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 3 * time.Second,
		Proxy: http.ProxyFromEnvironment,
		Dial: func(network, addr string) (net.Conn, error) {
			addr = strings.Split(addr, ":")[0]
			tmp := strings.Split(addr, "/")
			if len(tmp) != 2 {
				return nil, ErrInvalidService
			}
			return LoadBalance(network, tmp[0], tmp[1], reg)
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return func(w http.ResponseWriter, req *http.Request) {
		name, version, err := ExtractNameVersion(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reverseProxy := &httputil.ReverseProxy{
			Director: func(req1 *http.Request) {
				req1.URL.Scheme = "http"
				req1.URL.Host = name + "/" + version
			},
			Transport: transport,
			ErrorLog:  errorLog,
		}

		var handler http.Handler

		// TODO: make this more generic for any kind of hijacker.
		if IsWebsocket(req) {
			handler = websocketProxy(name, version, reg)
		} else {
			handler = reverseProxy
		}

		if middleware != nil {
			middleware(name, version, handler).ServeHTTP(w, req)
		} else {
			handler.ServeHTTP(w, req)
		}
	}
}
