package goproxy

import (
	"fmt"
	"log"
	"net/http"

	"github.com/creack/goproxy/registry"
)

// ServiceRegistry is a local registry of services/versions
var ServiceRegistry = registry.DefaultRegistry{
	"service1": {
		"v1": {
			"localhost:9091",
			"localhost:9092",
		},
	},
}

func Example() {
	http.HandleFunc("/", NewMultipleHostReverseProxy(ServiceRegistry))
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v\n", ServiceRegistry)
	})
	println("ready")
	log.Fatal(http.ListenAndServe(":9090", nil))
}
