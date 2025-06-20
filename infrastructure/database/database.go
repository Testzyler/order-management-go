package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var DatabasePool *pgxpool.Pool
var DBConfig = struct {
	Username       string
	Password       string
	Host           string
	Port           int
	DatabaseName   string
	DatabaseSchema string
}{
	Username:       viper.GetString("Database.Username"),
	Password:       viper.GetString("Database.Password"),
	Host:           viper.GetString("Database.Host"),
	Port:           viper.GetInt("Database.Port"),
	DatabaseName:   viper.GetString("Database.DatabaseName"),
	DatabaseSchema: viper.GetString("Database.DatabaseSchema"),
}

func InitializeDatabase() (*pgxpool.Pool, error) {
	// Ensure configuration is loaded
	userName := viper.GetString("Database.Username")
	password := viper.GetString("Database.Password")
	host := viper.GetString("Database.Host")
	port := viper.GetInt("Database.Port")
	databaseName := viper.GetString("Database.DatabaseName")
	databaseSchema := viper.GetString("Database.DatabaseSchema")

	// Log configuration for debugging (remove in production)
	fmt.Printf("Connecting to database at %s:%d...\n", host, port)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable&search_path=%s",
		userName, password, host, port, databaseName, databaseSchema,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := waitForDatabase(db, 30*time.Second); err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}

	db.Config().MaxConns = 500
	db.Config().MinIdleConns = 500
	db.Config().MaxConnLifetime = 5 * time.Minute

	fmt.Println("Database connection established successfully.")
	return db, nil
}

func NewDatabaseConnection() (*pgxpool.Pool, error) {
	if DatabasePool == nil {
		db, err := InitializeDatabase()
		if err != nil {
			return nil, fmt.Errorf("error initializing database: %w", err)
		}
		DatabasePool = db
	} else {
		fmt.Println("Using existing database connection.")
	}

	return DatabasePool, nil
}

func ShutdownDatabase() error {
	if DatabasePool != nil {
		DatabasePool.Close()
		fmt.Println("Database connection closed successfully.")
	}
	return nil
}

func waitForDatabase(pool *pgxpool.Pool, timeout time.Duration) error {
	log.Println("Waiting for database to be ready...")

	deadline := time.Now().Add(timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := pool.Ping(ctx)
		cancel()

		if err == nil {
			log.Println("Database is ready!")
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("database not ready after %s: %w", timeout, err)
		}

		log.Println("Database not ready, retrying in 1s...")
		time.Sleep(1 * time.Second)
	}
}
