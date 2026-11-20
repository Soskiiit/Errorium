package config

import (
	"fmt"
	"os"
	"strconv"
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
	Endpoint        string
	AccessKey       string
	SecretKey       string
	BucketName      string
	Region          string
	UseSSL          bool
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

type MinIOConfig = S3Config

type JWTConfig struct {
	SecretKey              string
	AccessTokenExpireMin   int
	RefreshTokenExpireDays int
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type SSOProviderConfig struct {
	ClientID     string
	ClientSecret string
}

type SSOConfig struct {
	RedirectBase string
	GitHub       SSOProviderConfig
	Google       SSOProviderConfig
	Yandex       SSOProviderConfig
}

type Config struct {
	Port             string
	SearchEngineAddr string
	DB               DBConfig
	S3               S3Config
	MinIO            S3Config
	JWT              JWTConfig
	SMTP             SMTPConfig
	SSO              SSOConfig
	FrontendURL      string
	AutoMigrate      bool
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func Load() *Config {
	accessKey := getEnv("S3_ACCESS_KEY", getEnv("GARAGE_ACCESS_KEY", getEnv("MINIO_ACCESS_KEY", "garageadmin")))
	secretKey := getEnv("S3_SECRET_KEY", getEnv("GARAGE_SECRET_KEY", getEnv("MINIO_SECRET_KEY", "garageadminpassword")))
	bucket := getEnv("S3_BUCKET_NAME", getEnv("GARAGE_BUCKET_NAME", getEnv("MINIO_BUCKET_NAME", "errorium-avatars")))

	s3Cfg := S3Config{
		Endpoint:        getEnv("S3_ENDPOINT", getEnv("GARAGE_ENDPOINT", getEnv("MINIO_ENDPOINT", "localhost:3900"))),
		AccessKey:       accessKey,
		SecretKey:       secretKey,
		BucketName:      bucket,
		Region:          getEnv("S3_REGION", getEnv("GARAGE_REGION", "garage")),
		UseSSL:          getEnv("S3_USE_SSL", getEnv("GARAGE_USE_SSL", getEnv("MINIO_USE_SSL", "false"))) == "true",
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Bucket:          bucket,
	}

	accessExpMin := 60
	if v := getEnv("ACCESS_TOKEN_EXPIRE_MINUTES", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			accessExpMin = n
		}
	}
	refreshExpDays := 30
	if v := getEnv("REFRESH_TOKEN_EXPIRE_DAYS", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			refreshExpDays = n
		}
	}

	searchHost := getEnv("SEARCH_ENGINE_HOST", "search_engine_core")
	searchPort := getEnv("SEARCH_ENGINE_PORT", "50051")
	searchEngineAddr := getEnv("SEARCH_ENGINE_ADDR", searchHost+":"+searchPort)

	smtpPort := 587
	if v := getEnv("SMTP_PORT", ""); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			smtpPort = n
		}
	}

	smtpCfg := SMTPConfig{
		Host:     getEnv("SMTP_HOST", ""),
		Port:     smtpPort,
		User:     getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", "no-reply@errorium.ru"),
	}

	ssoCfg := SSOConfig{
		RedirectBase: getEnv("SSO_REDIRECT_BASE", "http://localhost:8000"),
		GitHub: SSOProviderConfig{
			ClientID:     getEnv("GITHUB_CLIENT_ID", ""),
			ClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		},
		Google: SSOProviderConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
		Yandex: SSOProviderConfig{
			ClientID:     getEnv("YANDEX_CLIENT_ID", ""),
			ClientSecret: getEnv("YANDEX_CLIENT_SECRET", ""),
		},
	}

	return &Config{
		Port:             getEnv("GATEWAY_PORT", "8000"),
		SearchEngineAddr: searchEngineAddr,
		AutoMigrate:      getEnv("AUTO_MIGRATE", "true") == "true",
		FrontendURL:      getEnv("FRONTEND_URL", "http://localhost:3000"),
		SMTP:             smtpCfg,
		SSO:              ssoCfg,
		JWT: JWTConfig{
			SecretKey:              getEnv("JWT_SECRET_KEY", "change-this-in-production"),
			AccessTokenExpireMin:   accessExpMin,
			RefreshTokenExpireDays: refreshExpDays,
		},
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
