package main

import (
	"log"
	"os"
)

const defaultAdminKey = "admin12345"

type Config struct {
	Port     string
	DataDir  string
	AdminKey string
}

func loadConfigFromEnv() (Config, error) {
	cfg := Config{
		Port:     getenvDefault("PORT", "8765"),
		DataDir:  getenvDefault("OTA_DATA_DIR", "data"),
		AdminKey: getenvDefault("OTA_ADMIN_KEY", defaultAdminKey),
	}

	if cfg.AdminKey == defaultAdminKey {
		log.Println("AdminKey is defaulted to 'admin', please change your admin key")
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
