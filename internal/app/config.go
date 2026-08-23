package app

import "os"

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func LoadConfig() Config {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "./bunkerflow.db"
	}
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	return Config{DatabaseURL: databaseURL, HTTPAddr: addr}
}
