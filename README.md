# goproxy
GoProxy is a ReverseProxy / LoadBalancer helper for Golang

# Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/creack/goproxy"
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

func main() {
	http.HandleFunc("/", goproxy.NewMultipleHostReverseProxy(ServiceRegistry))
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v\n", ServiceRegistry)
	})
	println("ready")
	log.Fatal(http.ListenAndServe(":9090", nil))
}
```

# Limitations

Because we control only the connection, we can't have different http routes accross the same service.
For the same reason, we can't have both HTTP and HTTPS for the same service.
For now, this load balancer only supports HTTP.
