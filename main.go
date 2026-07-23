package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	db                      database.Client
	jwtSecret               string
	jwtIssuer               string
	jwtAudience             string
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

func readIntEnvOrDefault(envName string, fallback int) int {
	raw := os.Getenv(envName)
	if raw == "" {
		return fallback
	}

	v, err := strconv.Atoi(raw)
	if err != nil {
		log.Fatalf("%s has invalid integer %q: %v", envName, raw, err)
	}
	if v <= 0 {
		log.Fatalf("%s must be > 0, got %q", envName, raw)
	}

	return v
}

func main() {
	//.env file laden
	godotenv.Load(".env")

	pathToDB := os.Getenv("DB_PATH")
	if pathToDB == "" {
		log.Fatal("DB_URL must be set")
	}
	//db client erstellen
	db, err := database.NewClient(pathToDB)
	if err != nil {
		log.Fatalf("Couldn't connect to database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is not set")
	}

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "dienstleistung-api"
	}

	jwtAudience := os.Getenv("JWT_AUDIENCE")
	if jwtAudience == "" {
		jwtAudience = "dienstleistung-api-users"
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

	// .env variabeln standart werte wenn nicht vorhanden
	accessTokenTTL := readDurationEnvOrDefault("ACCESS_TOKEN_TTL", time.Hour*24)
	refreshTokenTTL := readDurationEnvOrDefault("REFRESH_TOKEN_TTL", time.Hour*24*7)
	refreshedAccessTokenTTL := readDurationEnvOrDefault("REFRESH_ACCESS_TOKEN_TTL", time.Hour)
	loginRateLimitPerMinute := readIntEnvOrDefault("LOGIN_RATE_LIMIT_PER_MINUTE", 10)
	loginFailedRateLimitPerMinute := readIntEnvOrDefault("LOGIN_FAILED_RATE_LIMIT_PER_MINUTE", 5)
	refreshRateLimitPerMinute := readIntEnvOrDefault("REFRESH_RATE_LIMIT_PER_MINUTE", 30)

	cfg := apiConfig{
		db:                      db,
		jwtSecret:               jwtSecret,
		jwtIssuer:               jwtIssuer,
		jwtAudience:             jwtAudience,
		platform:                platform,
		filepathRoot:            filepathRoot,
		port:                    port,
		accessTokenTTL:          accessTokenTTL,
		refreshTokenTTL:         refreshTokenTTL,
		refreshedAccessTokenTTL: refreshedAccessTokenTTL,
	}
	//datenbank mit test daten seeden wenn keine vorhanden sind
	if err := cfg.ensureDatabaseSeeded(); err != nil {
		log.Fatalf("Couldn't ensure default database seed data: %v", err)
	}

	mux := http.NewServeMux()
	// file server für das frontend
	appHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/", appHandler)
	// login/token refresh limiters
	loginLimiter := newFixedWindowRateLimiter(time.Minute, loginRateLimitPerMinute)
	loginFailureLimiter := newFixedWindowRateLimiter(time.Minute, loginFailedRateLimitPerMinute)
	refreshLimiter := newFixedWindowRateLimiter(time.Minute, refreshRateLimitPerMinute)

	//api auth endpoints
	mux.HandleFunc("POST /api/login", rateLimitByIP("login", loginLimiter, rateLimitFailedLoginsByIP("login_failed", loginFailureLimiter, cfg.handlerLogin)))
	mux.HandleFunc("POST /api/refresh", rateLimitByIP("refresh", refreshLimiter, cfg.handlerRefresh))
	mux.HandleFunc("POST /api/revoke", cfg.handlerRevoke)
	//api endpoints
	mux.HandleFunc("POST /api/users", cfg.handlerUsersCreate)
	mux.HandleFunc("GET /api/appointments", cfg.handlerAppointmentsList)
	mux.HandleFunc("POST /api/appointments", cfg.handlerAppointmentsCreate)
	mux.HandleFunc("PUT /api/appointments/{id}", cfg.handlerAppointmentsUpdate)
	mux.HandleFunc("DELETE /api/appointments/{id}", cfg.handlerAppointmentsCancel)
	mux.HandleFunc("GET /api/availability", cfg.handlerAvailabilityGet)
	mux.HandleFunc("POST /api/availability", cfg.handlerAvailabilityCreate)
	mux.HandleFunc("GET /api/employees", cfg.handlerEmployeesList)
	mux.HandleFunc("POST /api/employees/resolve", cfg.handlerEmployeesResolve)
	mux.HandleFunc("GET /api/services/tree", cfg.handlerServicesTree)
	//api test endpoints
	mux.HandleFunc("POST /api/test/reset-and-seed", cfg.handlerTestResetAndSeed)
	mux.HandleFunc("DELETE /api/appointments/delete", cfg.handlerAppointmentsCancelAll)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving on: http://localhost:%s/app/\n", port)
	//server starten
	log.Fatal(srv.ListenAndServe())
}
