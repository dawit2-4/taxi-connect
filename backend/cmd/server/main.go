package main

import (
	"context"

	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"taxi-connect/backend/internal/auth"
	"taxi-connect/backend/internal/config"
	"taxi-connect/backend/internal/queue"
	queuerepo "taxi-connect/backend/internal/queue"
	"taxi-connect/backend/internal/terminals"
	terminalrepo "taxi-connect/backend/internal/terminals/repository"
	userRepo "taxi-connect/backend/internal/user"

	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver with database/sql
	"github.com/google/uuid"
)

// userRepositoryAdapter adapts the user package's UserRepository to the auth package's UserRepository interface.
type userRepositoryAdapter struct {
	userRepo userRepo.UserRepository
}

// GetUserByID returns the user for the given ID.
// This method satisfies the auth.UserRepository interface.
func (a *userRepositoryAdapter) GetUserByID(ctx context.Context, id uuid.UUID) (interface{}, error) {
	// Call the actual user repository
	_, err := a.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// We only need to verify that the user exists, so we can return any non-nil value
	// The auth middleware only checks for error, not the actual user object
	return struct{}{}, nil
}

func main() {
	// Load database configuration and connect
	dbConfig := config.LoadDBConfig()
	db, err := config.ConnectDB(dbConfig)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	// Initialize terminal repository, service, and handler
	terminalRepo := terminalrepo.NewPostgresTerminalRepository(db)
	terminalService := terminals.NewTerminalService(terminalRepo)
	terminalHandler := terminals.NewTerminalHandler(terminalService)

	// Initialize queue repository
	queueRepo := queuerepo.NewPostgresQueueRepository(db)

	// Initialize user repository and service
	userRepoInstance := userRepo.NewPostgresUserRepository(db)
	_ = userRepo.NewUserService(userRepoInstance) // Service initialized but not used in main

	// Create connection manager for WebSocket subscriptions
	connectionManager := queue.NewConnectionManager()

	// Create notifier that uses the connection manager
	notifier := &queue.SimpleQueueNotifier{
		ConnMgr: connectionManager,
	}

	// Initialize queue service with notifier
	queueService := queue.NewQueueService(queueRepo, userRepoInstance, terminalRepo, notifier)
	queueHandler := queue.NewQueueHandler(queueService, connectionManager)

	// Load JWT secret from config (in production, this should come from environment variables)
	jwtSecret := []byte("your-secret-key-change-in-production")

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Terminal routes
	mux.HandleFunc("/terminals", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			terminalHandler.ListTerminalsHandler(w, r)
		case http.MethodPost:
			terminalHandler.CreateTerminalHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/terminals/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			terminalHandler.GetTerminalHandler(w, r)
		case http.MethodPut:
			terminalHandler.UpdateTerminalHandler(w, r)
		case http.MethodDelete:
			terminalHandler.DeleteTerminalHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/terminals/name/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			terminalHandler.GetTerminalByNameHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/terminals/near", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			terminalHandler.FindTerminalsNearHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Queue routes
	var userAuthRepo auth.UserRepository = &userRepositoryAdapter{userRepoInstance}
	jwtMiddleware := auth.JWTMiddleware(userAuthRepo, jwtSecret)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			queueHandler.StoreReportHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/queue/report", jwtMiddleware(handler))

	mux.HandleFunc("/queue/estimate", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			queueHandler.GetEstimateHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// WebSocket routes
	mux.HandleFunc("/ws/queue/updates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			queueHandler.QueueWebSocketHandler(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

// runMigrations executes all SQL migration files in order
func runMigrations(db *sql.DB) error {
	files := []string{
		"migrations/001_create_extensions.sql",
		"migrations/002_create_tables.sql",
		"migrations/003_create_queue_table.sql",
	}

	for _, file := range files {
		log.Printf("Running migration: %s", file)
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("executing migration %s: %w", file, err)
		}
		log.Printf("Completed migration: %s", file)
	}
	return nil
}