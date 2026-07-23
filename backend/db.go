package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func connectDB() *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/aicam?sslmode=disable"
	}

	var db *sql.DB
	var err error

	// retry: utile quando il backend parte prima che Postgres sia pronto (docker-compose)
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				break
			} else {
				err = pingErr
			}
		}
		log.Printf("DB non ancora pronto (tentativo %d/10): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatalf("impossibile connettersi al database: %v", err)
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)

	fmt.Println("Connesso a PostgreSQL")
	return db
}
