package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	multiplexer.HandleFunc("GET /api/healthz", readinessHandler)
	multiplexer.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	multiplexer.HandleFunc("POST /admin/reset", apiCfg.metricsResetHandler)
	multiplexer.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
}

func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (a *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	htmlBody := fmt.Sprintf("<html><body><h1>Welcome, Chirpy Admin</h1><p>Chirpy has been visited %d times!</p></body></html>", a.fileserverHits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(htmlBody))
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

func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	type postData struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	pData := postData{}
	err := decoder.Decode(&pData)
	if err != nil {
		statuscode := 500
		validateErrorHelper(w, r, err, statuscode)
		return
	}

	if len(pData.Body) > 140 {
		statuscode := 400
		validateErrorHelper(w, r, fmt.Errorf("Chirp is too long"), statuscode)
		return
	} else {
		respStatus := 200
		validateSuccessHelper(w, r, pData.Body, respStatus)
	}
}

func validateErrorHelper(w http.ResponseWriter, r *http.Request, err error, respStatus int) {
	type postErrorResponse struct {
		Error string `json:"error"`
	}
	respBody := postErrorResponse{
		Error: err.Error(),
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	w.Write(dat)
}

func validateSuccessHelper(w http.ResponseWriter, r *http.Request, text string, respStatus int) {
	type postSuccessResponse struct {
		Cleaned_body string `json:"cleaned_body"`
	}
	cleanedText := validateProfanityHelper(text)

	respBody := postSuccessResponse{
		Cleaned_body: cleanedText,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	w.Write(dat)
}

func validateProfanityHelper(text string) string {
	profanitys := []string{"kerfuffle", "sharbert", "fornax"}

	splitText := strings.Fields(text)
	for i, word := range splitText {
		for _, profanity := range profanitys {
			if strings.ToLower(word) == profanity {
				splitText[i] = "****"
			}
		}
	}
	filteredText := strings.Join(splitText, " ")
	return filteredText
}
