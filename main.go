package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	multiplexer := http.NewServeMux()
	var apiCfg apiConfig

	setupHandlers(multiplexer, &apiCfg)

	var server http.Server
	server.Handler = multiplexer
	server.Addr = ":8080"

	var system http.FileSystem
	url := http.Dir(".")
	system = url

	indexServer := http.FileServer(system)

	multiplexer.Handle("/app/", (&apiCfg).MetricsInc(http.StripPrefix("/app", indexServer)))
	server.ListenAndServe()

}

func setupHandlers(multiplexer *http.ServeMux, apiCfg *apiConfig) {
	multiplexer.HandleFunc("GET /healthz", readinessHandler)
	multiplexer.HandleFunc("GET /metrics", apiCfg.metricsHandler)
	multiplexer.HandleFunc("POST /reset", apiCfg.metricsResetHandler)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (a *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	body := fmt.Sprintf("Hits: %v", a.fileserverHits.Load())
	w.Write([]byte(body))
}

func (a *apiConfig) metricsResetHandler(w http.ResponseWriter, r *http.Request) {
	a.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (a *apiConfig) MetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
