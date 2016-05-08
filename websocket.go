package goproxy

import (
	"io"
	"log"
	"net/http"

	"github.com/creack/goproxy/registry"
)

func websocketProxy(name, version string, reg registry.Registry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		targetConn, err := LoadBalance("tcp", name, version, reg)
		if err != nil {
			http.Error(w, "Destination not reachable.", http.StatusInternalServerError)
			return
		}
		defer targetConn.Close()

		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Invalid connection type. Can't hijack.", http.StatusInternalServerError)
			return
		}
		sourceConn, _, err := hijacker.Hijack()
		if err != nil {
			log.Printf("Hijack error: %v", err)
			return
		}
		defer sourceConn.Close()

		// Write the initial request to the target (Connection & Upgrade headers).
		if err := req.Write(targetConn); err != nil {
			log.Printf("Error copying request to target: %s", err)
			return
		}

		ch := make(chan error, 2)
		go func() { _, _ = io.Copy(targetConn, sourceConn); _ = targetConn.Close(); ch <- nil }()
		go func() { _, _ = io.Copy(sourceConn, targetConn); _ = sourceConn.Close(); ch <- nil }()
		<-ch
		<-ch
		close(ch)
	})
}
