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

type S3Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	UseSSL     bool
}

type MinIOConfig = S3Config

type Config struct {
	Port        string
	DB          DBConfig
	S3          S3Config
	MinIO       S3Config
	AutoMigrate bool
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func Load() *Config {
	s3Cfg := S3Config{
		Endpoint:   getEnv("S3_ENDPOINT", getEnv("GARAGE_ENDPOINT", getEnv("MINIO_ENDPOINT", "localhost:3900"))),
		AccessKey:  getEnv("S3_ACCESS_KEY", getEnv("GARAGE_ACCESS_KEY", getEnv("MINIO_ACCESS_KEY", "garageadmin"))),
		SecretKey:  getEnv("S3_SECRET_KEY", getEnv("GARAGE_SECRET_KEY", getEnv("MINIO_SECRET_KEY", "garageadminpassword"))),
		BucketName: getEnv("S3_BUCKET_NAME", getEnv("GARAGE_BUCKET_NAME", getEnv("MINIO_BUCKET_NAME", "errorium-avatars"))),
		Region:     getEnv("S3_REGION", getEnv("GARAGE_REGION", "garage")),
		UseSSL:     getEnv("S3_USE_SSL", getEnv("GARAGE_USE_SSL", getEnv("MINIO_USE_SSL", "false"))) == "true",
	}

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
		S3:    s3Cfg,
		MinIO: s3Cfg,
	}
}
