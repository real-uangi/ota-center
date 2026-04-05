package main

import (
	"errors"
	"os"
)

type Config struct {
	Port     string
	DataDir  string
	AdminKey string
}

func loadConfigFromEnv() (Config, error) {
	cfg := Config{
		Port:     getenvDefault("PORT", "8080"),
		DataDir:  getenvDefault("OTA_DATA_DIR", "data"),
		AdminKey: os.Getenv("OTA_ADMIN_KEY"),
	}

	if cfg.AdminKey == "" {
		return Config{}, errors.New("OTA_ADMIN_KEY is required")
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
