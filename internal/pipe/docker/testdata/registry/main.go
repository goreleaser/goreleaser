package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

func main() {
	listener, err := net.Listen("tcp", ":5000")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("ready")
	log.Fatal(http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/":
			fmt.Fprint(w, "{}")
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, `{"errors":[{"code":"DENIED","message":"test registry denied this push"}]}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
}
