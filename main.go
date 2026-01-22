package main

import "net/http"

func main() {
	multiplexer := http.NewServeMux()
	var server http.Server
	server.Handler = multiplexer
	server.Addr = ":8080"

	var system http.FileSystem
	url := http.Dir(".")
	system = url

	indexServer := http.FileServer(system)

	multiplexer.Handle("/", indexServer)
	server.ListenAndServe()

}
