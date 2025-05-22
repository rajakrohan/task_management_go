package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"task-management-service/internal/api"
	"task-management-service/internal/repository"
	"task-management-service/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:root123@localhost:5432/user?sslmode=disable"
	}

	log.Printf("Connecting to database: %s", dbURL)
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database")

	// Initialize Redis client
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	log.Printf("Connecting to Redis at %s:%s", redisHost, redisPort)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Successfully connected to Redis")

	// Create and verify database tables
	log.Println("Initializing database schema...")
	if err := initializeDatabase(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	log.Println("Database schema initialized successfully")

	// Initialize dependencies
	taskRepo := repository.NewPostgresTaskRepository(db, rdb)
	taskService := service.NewTaskService(taskRepo)
	taskHandler := api.NewTaskHandler(taskService)

	// Initialize router
	router := gin.Default()

	// Register routes
	taskHandler.RegisterRoutes(router)

	// Add health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func initializeDatabase(db *sql.DB) error {
	// Drop existing table if it exists (for development only)
	// In production, you should use proper migrations
	_, err := db.Exec(`
		DROP TABLE IF EXISTS tasks;
	`)
	if err != nil {
		return fmt.Errorf("failed to drop existing table: %w", err)
	}

	// Create tasks table
	_, err = db.Exec(`
		CREATE TABLE tasks (
			id UUID PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create tasks table: %w", err)
	}

	// Verify table structure
	rows, err := db.Query(`
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'tasks'
		ORDER BY ordinal_position;
	`)
	if err != nil {
		return fmt.Errorf("failed to verify table structure: %w", err)
	}
	defer rows.Close()

	log.Println("Verifying table structure:")
	for rows.Next() {
		var columnName, dataType, isNullable string
		if err := rows.Scan(&columnName, &dataType, &isNullable); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		log.Printf("Column: %s, Type: %s, Nullable: %s", columnName, dataType, isNullable)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating table structure: %w", err)
	}

	return nil
}
