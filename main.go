package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	db                      database.Client
	jwtSecret               string
	platform                string
	filepathRoot            string
	port                    string
	accessTokenTTL          time.Duration
	refreshTokenTTL         time.Duration
	refreshedAccessTokenTTL time.Duration
}

func readDurationEnvOrDefault(envName string, fallback time.Duration) time.Duration {
	raw := os.Getenv(envName)
	if raw == "" {
		return fallback
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Fatalf("%s has invalid duration %q: %v", envName, raw, err)
	}
	if d <= 0 {
		log.Fatalf("%s must be > 0, got %q", envName, raw)
	}

	return d
}

func main() {
	godotenv.Load(".env")

	pathToDB := os.Getenv("DB_PATH")
	if pathToDB == "" {
		log.Fatal("DB_URL must be set")
	}

	db, err := database.NewClient(pathToDB)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("PLATFORM environment variable is not set")
	}

	filepathRoot := os.Getenv("FILEPATH_ROOT")
	if filepathRoot == "" {
		log.Fatal("FILEPATH_ROOT environment variable is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}

	accessTokenTTL := readDurationEnvOrDefault("ACCESS_TOKEN_TTL", time.Hour*24*30)
	refreshTokenTTL := readDurationEnvOrDefault("REFRESH_TOKEN_TTL", time.Hour*24*60)
	refreshedAccessTokenTTL := readDurationEnvOrDefault("REFRESH_ACCESS_TOKEN_TTL", time.Hour)

	cfg := apiConfig{
		db:                      db,
		jwtSecret:               jwtSecret,
		platform:                platform,
		filepathRoot:            filepathRoot,
		port:                    port,
		accessTokenTTL:          accessTokenTTL,
		refreshTokenTTL:         refreshTokenTTL,
		refreshedAccessTokenTTL: refreshedAccessTokenTTL,
	}

	mux := http.NewServeMux()
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", appHandler)

	mux.HandleFunc("POST /api/login", cfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", cfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)

	mux.HandleFunc("POST /api/users", cfg.handlerUsersCreate)
	mux.HandleFunc("GET /api/appointments", cfg.handlerAppointmentsList)
	mux.HandleFunc("POST /api/appointments", cfg.handlerAppointmentsCreate)
	mux.HandleFunc("DELETE /api/appointments/{id}", cfg.handlerAppointmentsCancel)
	mux.HandleFunc("GET /api/availability", cfg.handlerAvailabilityGet)
	mux.HandleFunc("POST /api/availability", cfg.handlerAvailabilityCreate)
	mux.HandleFunc("GET /api/employees", cfg.handlerEmployeesList)
	mux.HandleFunc("POST /api/employees/resolve", cfg.handlerEmployeesResolve)
	mux.HandleFunc("POST /api/test/reset-and-seed", cfg.handlerTestResetAndSeed)
	mux.HandleFunc("DELETE /api/appointments/delete", cfg.handlerAppointmentsCancelAll)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on: http://localhost:%s/app/\n", port)
	log.Fatal(srv.ListenAndServe())
}
