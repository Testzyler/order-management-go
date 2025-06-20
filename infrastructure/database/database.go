package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

var DatabasePool *sql.DB
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

func InitializeDatabase() (*sql.DB, error) {
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

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(100)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(time.Duration(180) * time.Second)

	fmt.Println("Database connection established successfully.")
	return db, nil
}

func NewDatabaseConnection() (*sql.DB, error) {
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
		if err := DatabasePool.Close(); err != nil {
			fmt.Printf("Error closing database connection: %v\n", err)
			return err
		} else {
			fmt.Println("Database connection closed successfully.")
			return nil
		}
	}
	return nil
}
