package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
	platform       string
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("error opening database: %v", err)
		return
	}
	multiplexer := http.NewServeMux()
	var apiCfg apiConfig

	dbQueries := database.New(db)
	apiCfg.queries = dbQueries
	apiCfg.platform = os.Getenv("PLATFORM")
	setupHandlers(multiplexer, &apiCfg)

	var server http.Server
	server.Handler = multiplexer
	server.Addr = ":8080"

	var system http.FileSystem
	url := http.Dir(".")
	system = url

	indexServer := http.FileServer(system)

	multiplexer.Handle("/app/", (&apiCfg).MetricsInc(http.StripPrefix("/app/", indexServer)))
	server.ListenAndServe()

}

func setupHandlers(multiplexer *http.ServeMux, apiCfg *apiConfig) {
	multiplexer.HandleFunc("GET /api/healthz", readinessHandler)
	multiplexer.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	multiplexer.HandleFunc("POST /admin/reset", apiCfg.metricsResetHandler)
	//multiplexer.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	multiplexer.HandleFunc("POST /api/users", apiCfg.usershandler)
	multiplexer.HandleFunc("POST /api/chirps", apiCfg.chirpsHandler)
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
	if a.platform != "dev" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("403 Forbidden"))
		return
	}
	err := a.queries.ClearUsers(r.Context())
	if err != nil {
		log.Printf("Error Clearing Users: %s", err)
		w.WriteHeader(500)
		return
	}
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

func ErrorHelper(w http.ResponseWriter, r *http.Request, err error, respStatus int) {
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

func (a *apiConfig) usershandler(w http.ResponseWriter, r *http.Request) {
	type jsonRequest struct {
		Email string `json:"email"`
	}

	type jsonResponse struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}

	var request jsonRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request)
	if err != nil {
		statusCode := 400
		ErrorHelper(w, r, err, statusCode)
		return
	}
	user, err := a.queries.CreateUser(r.Context(), request.Email)
	if err != nil {
		statusCode := 500
		ErrorHelper(w, r, err, statusCode)
		return
	}

	response := jsonResponse{
		Id:         user.ID,
		Created_at: user.CreatedAt,
		Updated_at: user.UpdatedAt,
		Email:      user.Email,
	}
	dat, err := json.Marshal(response)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}

	respStatus := 201
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	w.Write(dat)

}

func (a *apiConfig) chirpsHandler(w http.ResponseWriter, r *http.Request) {
	type jsonPayload struct {
		Body    string    `json:"body"`
		User_id uuid.UUID `json:"user_id"`
	}
	type Chirp struct {
		Id         uuid.UUID `json:"id"`
		Created_at time.Time `json:"created_at"`
		Updated_at time.Time `json:"updated_at"`
		Body       string    `json:"body"`
		User_id    uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	pData := jsonPayload{}
	err := decoder.Decode(&pData)
	if err != nil {
		statuscode := 500
		ErrorHelper(w, r, err, statuscode)
		return
	}

	if len(pData.Body) > 140 {
		statuscode := 400
		ErrorHelper(w, r, fmt.Errorf("Chirp is too long"), statuscode)
		return
	}

	databaseChirp, err := a.queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   validateProfanityHelper(pData.Body),
		UserID: pData.User_id,
	})
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}

	newChirp := Chirp{
		Id:         databaseChirp.ID,
		Created_at: databaseChirp.CreatedAt,
		Updated_at: databaseChirp.UpdatedAt,
		Body:       databaseChirp.Body,
		User_id:    databaseChirp.UserID,
	}

	dat, err := json.Marshal(newChirp)
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(dat)
}

/* func validateSuccessHelper(w http.ResponseWriter, r *http.Request, text string, respStatus int) {
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

*/
