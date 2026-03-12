package main

import (
	"chirpy/internal/auth"
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
	jwtSecret      string
}
type Chirp struct {
	ID         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Body       string    `json:"body"`
	User_id    uuid.UUID `json:"user_id"`
}

type userRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type userResponse struct {
	Id         uuid.UUID `json:"id"`
	Created_at time.Time `json:"created_at"`
	Updated_at time.Time `json:"updated_at"`
	Email      string    `json:"email"`
	Token      string    `json:"token"`
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
	apiCfg.jwtSecret = os.Getenv("KC8I7S97uxNLnMoUb676+olJNQ/ONWHKbL0e3aoc8Iau8NwGrtodFzS5pPZK1FkgcHX99ZmVdpdMHpuLfdqZuA==")
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
	multiplexer.HandleFunc("POST /api/users", apiCfg.usershandler)
	multiplexer.HandleFunc("POST /api/chirps", apiCfg.chirpsHandler)
	multiplexer.HandleFunc("GET /api/chirps", apiCfg.getchirpsHandler)
	multiplexer.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)
	multiplexer.HandleFunc("POST /api/login", apiCfg.loginHandler)
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

	var request userRequest

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

	hash, err := auth.HashPassword(request.Password)
	if err != nil {
		log.Printf("Error hashing password: %s", err)
		w.WriteHeader(500)
		return
	}

	err = a.queries.AddPassword(r.Context(), database.AddPasswordParams{
		ID:             user.ID,
		HashedPassword: hash,
	})
	if err != nil {
		log.Printf("Error storing password: %s", err)
		w.WriteHeader(500)
		return
	}

	response := userResponse{
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

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		statusCode := 401
		ErrorHelper(w, r, err, statusCode)
		return
	}
	tokenUserID, err := auth.ValidateJWT(bearerToken, a.jwtSecret)
	if err != nil {
		statusCode := 401
		ErrorHelper(w, r, err, statusCode)
		return
	}

	decoder := json.NewDecoder(r.Body)
	pData := jsonPayload{}
	err = decoder.Decode(&pData)
	if err != nil {
		statuscode := 500
		ErrorHelper(w, r, err, statuscode)
		return
	}

	/*

		fmt.Println("provided id")
		fmt.Println(pData.User_id)
		fmt.Print("\ntokenid\n")
		fmt.Println(tokenUserID)
		fmt.Print("\n")

		if pData.User_id != tokenUserID {
			err = fmt.Errorf("Token user ID doesn't match provided User")
			statuscode := 401
			ErrorHelper(w, r, err, statuscode)
			return
		}

	*/

	if len(pData.Body) > 140 {
		statuscode := 400
		ErrorHelper(w, r, fmt.Errorf("Chirp is too long"), statuscode)
		return
	}

	databaseChirp, err := a.queries.CreateChirp(r.Context(), database.CreateChirpParams{
		Body: validateProfanityHelper(pData.Body),
		//UserID: pData.User_id,
		UserID: tokenUserID,
	})
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}

	newChirp := Chirp{
		ID:         databaseChirp.ID,
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

func (a *apiConfig) getchirpsHandler(w http.ResponseWriter, r *http.Request) {
	allChirps, err := a.queries.GetAllChirps(r.Context())
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	chirpsArray := make([]Chirp, len(allChirps))
	for i, chirp := range allChirps {
		chirpsArray[i] = Chirp{
			ID:         chirp.ID,
			Created_at: chirp.CreatedAt,
			Updated_at: chirp.UpdatedAt,
			Body:       chirp.Body,
			User_id:    chirp.UserID,
		}
	}

	dat, err := json.Marshal(chirpsArray)
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)

}

func (a *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	requestUserID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		statuscode := (404)
		ErrorHelper(w, r, err, statuscode)
		return
	}

	dbChirp, err := a.queries.GetChirpbyID(r.Context(), requestUserID)
	if err != nil {
		statuscode := (404)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	jsonChirp := Chirp{
		ID:         dbChirp.ID,
		Created_at: dbChirp.CreatedAt,
		Updated_at: dbChirp.UpdatedAt,
		Body:       dbChirp.Body,
		User_id:    dbChirp.UserID,
	}

	dat, err := json.Marshal(jsonChirp)
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)

}

func (a *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type userRequest struct {
		Password           string `json:"password"`
		Email              string `json:"email"`
		Expires_in_seconds int    `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	pData := userRequest{}
	err := decoder.Decode(&pData)
	if err != nil {
		statusCode := 400
		ErrorHelper(w, r, err, statusCode)
		return
	}
	user, err := a.queries.GetPassword(r.Context(), pData.Email)
	if err != nil {
		statuscode := (401)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	ok, err := auth.CheckPasswordHash(pData.Password, user.HashedPassword)
	if err != nil {
		statuscode := (401)
		ErrorHelper(w, r, err, statuscode)
		return
	}

	if !ok {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Incorrect email or password"))
		return
	}

	expires := time.Hour
	if pData.Expires_in_seconds != 0 {
		expires = time.Duration(pData.Expires_in_seconds) * time.Second
	}

	token, err := auth.MakeJWT(user.ID, a.jwtSecret, expires)
	if err != nil {
		statuscode := (401)
		ErrorHelper(w, r, err, statuscode)
	}

	userStruct := userResponse{
		Id:         user.ID,
		Created_at: user.CreatedAt,
		Updated_at: user.UpdatedAt,
		Email:      user.Email,
		Token:      token,
	}

	dat, err := json.Marshal(userStruct)
	if err != nil {
		statuscode := (500)
		ErrorHelper(w, r, err, statuscode)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(dat)

}
