package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"taxi-connect/backend/internal/config"
	"taxi-connect/backend/internal/queue"
	queuerepo "taxi-connect/backend/internal/queue"
	"taxi-connect/backend/internal/terminals"
	terminalrepo "taxi-connect/backend/internal/terminals/repository"

	_ "github.com/jackc/pgx/v5/stdlib" // Register pgx driver with database/sql
)

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

	// Initialize queue repository, service, and handler
	queueRepo := queuerepo.NewPostgresQueueRepository(db)
	queueService := queue.NewQueueService(queueRepo, terminalRepo)
	queueHandler := queue.NewQueueHandler(queueService)

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
	mux.HandleFunc("/queue/report", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			queueHandler.StoreReportHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/queue/estimate", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			queueHandler.GetEstimateHandler(w, r)
		default:
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