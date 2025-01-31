// Main application runner.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/jackc/pgx/v5/pgxpool"

	"event-reservation-api/db"
	"event-reservation-api/middlewares"
	"event-reservation-api/routes"
)

// Populate the database with initial data if the populate flag is set.
func populateDatabase(populateFlag *bool, pool *pgxpool.Pool) {
	if *populateFlag {
		// if the flag is provided...
		fmt.Println("Populating the database with fake data and adding admin user...")
		err := db.PopulateDatabase(pool)
		if err != nil {
			log.Fatalf("Failed to populate the database: %v\n", err)
		}
		fmt.Println("Database populated successfully.")

	} else {
		// if the flag is not provided...
		fmt.Println("Skipping database population, adding only admin user...")
		err := db.AddAdminUser(nil, pool)
		if err != nil {
			log.Fatalf("Failed to populate the database: %v\n", err)
		}

	}
}

//	@title			Ticket Reservation API
//	@version		1.0
//	@description	Simple API for managing ticket reservations.

//	@host		localhost:8080
//	@BasePath	/api/

// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	// Initialize the JWT secret.
	jwtSecret := middlewares.InitJWTSecret()

	// Parse the command line flags.
	populateFlag := flag.Bool("populate", false, "Populate the database with initial data.")
	flag.Parse()

	// Get the connection pool.
	pool, err := db.Connect()
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// Populate the database if the flag is set.
	populateDatabase(populateFlag, pool)

	// Set up the API routes.
	r := routes.SetupRoutes(pool, jwtSecret)

	// Start the server (defaults to port 8080).
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	// Enable CORS.
	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Start goroutine to clean up expired tokens.
	middlewares.StartTokenCleanupTask(pool, time.Hour)

	// Log the server start.
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, cors(r)))
}
