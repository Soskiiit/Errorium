package config

import (
	"fmt"
	"os"
)

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

type MinIOConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	UseSSL     bool
}

type Config struct {
	Port        string
	DB          DBConfig
	MinIO       MinIOConfig
	AutoMigrate bool
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func Load() *Config {
	return &Config{
		Port:        getEnv("GATEWAY_PORT", "8000"),
		AutoMigrate: getEnv("AUTO_MIGRATE", "true") == "true",
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			Name:     getEnv("DB_NAME", "errorium"),
			User:     getEnv("DB_USER", "errorium_user"),
			Password: getEnv("DB_PASSWORD", "errorium_password"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		MinIO: MinIOConfig{
			Endpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey:  getEnv("MINIO_SECRET_KEY", "minioadminpassword"),
			BucketName: getEnv("MINIO_BUCKET_NAME", "errorium-avatars"),
			UseSSL:     getEnv("MINIO_USE_SSL", "false") == "true",
		},
	}
}
