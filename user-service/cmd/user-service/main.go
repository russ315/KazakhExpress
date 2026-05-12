package main

import (
	"log"
	"net/http"
	"os"

	"kazakhexpress/user-service/internal/email"
	httpapi "kazakhexpress/user-service/internal/http"
	"kazakhexpress/user-service/internal/user"
)

func main() {
	port := os.Getenv("USER_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/kazakhexpress_users?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	repo, err := user.NewPostgresRepository(dbURL)
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	smtpEmailService := email.NewSMTPEmailService()
	service := user.NewService(repo, jwtSecret, smtpEmailService)
	handler := httpapi.NewHandler(service)

	log.Printf("user service started on :%s", port)
	if err := http.ListenAndServe(":"+port, handler.Routes()); err != nil {
		log.Fatal(err)
	}
}
