package configs

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var DB *sqlx.DB

func SetupDatabase(appName *string) *sqlx.DB {

	appNameVal := "default"
	if appName != nil && *appName != "" {
		appNameVal = *appName
	}
	// _ = godotenv.Load(".env")
	// Load .env
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	schema := os.Getenv("DB_SCHEMA")

	maxOpenConns := atoiDefault(os.Getenv("DB_MAX_OPEN_CONNS"), 2)
	maxIdleConns := atoiDefault(os.Getenv("DB_MAX_IDLE_CONNS"), 2)
	connMaxLifetimeMin := atoiDefault(os.Getenv("DB_CONN_MAX_LIFETIME_MIN"), 30)
	connMaxIdleTimeMin := atoiDefault(os.Getenv("DB_CONN_MAX_IDLE_TIME_MIN"), 10)

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable application_name=%s options='-c search_path=%s'",
		host, port, user, password, dbname, appNameVal, schema,
	)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil
	}

	sqlDB := db.DB
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)
	// Set connection maxlifetime
	sqlDB.SetConnMaxIdleTime(time.Duration(connMaxIdleTimeMin) * time.Minute)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetimeMin) * time.Minute)

	if err = db.Ping(); err != nil {
		log.Printf("DB not reachable at startup: %v", err)
		return db
	}

	log.Printf("Connected to DB (%s:%s) app=%s", host, port, appNameVal)
	DB = db
	return db
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
