package config_test

import (
	"os"
	"testing"

	"github.com/soskiiit/errorium/services/gateway/pkg/config"
)

func TestLoadDefaults(t *testing.T) {
	os.Clearenv()

	cfg := config.Load()
	if cfg.Port != "8000" {
		t.Errorf("expected port 8000, got %s", cfg.Port)
	}
	if cfg.S3.Endpoint != "localhost:3900" {
		t.Errorf("expected S3 endpoint localhost:3900, got %s", cfg.S3.Endpoint)
	}
	if cfg.S3.AccessKey != "garageadmin" {
		t.Errorf("expected AccessKey garageadmin, got %s", cfg.S3.AccessKey)
	}
	if cfg.S3.BucketName != "errorium-avatars" {
		t.Errorf("expected BucketName errorium-avatars, got %s", cfg.S3.BucketName)
	}
	if cfg.S3.Region != "garage" {
		t.Errorf("expected Region garage, got %s", cfg.S3.Region)
	}
	if cfg.MinIO.Endpoint != cfg.S3.Endpoint {
		t.Errorf("expected MinIO alias to match S3 endpoint")
	}
	if cfg.SearchEngineAddr != "search_engine_core:50051" {
		t.Errorf("expected SearchEngineAddr search_engine_core:50051, got %s", cfg.SearchEngineAddr)
	}
}

func TestLoadSearchEngineAddrOverride(t *testing.T) {
	os.Clearenv()
	t.Setenv("SEARCH_ENGINE_ADDR", "custom_engine:9090")

	cfg := config.Load()
	if cfg.SearchEngineAddr != "custom_engine:9090" {
		t.Errorf("expected SearchEngineAddr custom_engine:9090, got %s", cfg.SearchEngineAddr)
	}
}

func TestLoadSearchEngineHostPortFallback(t *testing.T) {
	os.Clearenv()
	t.Setenv("SEARCH_ENGINE_HOST", "my_engine")
	t.Setenv("SEARCH_ENGINE_PORT", "7000")

	cfg := config.Load()
	if cfg.SearchEngineAddr != "my_engine:7000" {
		t.Errorf("expected SearchEngineAddr my_engine:7000, got %s", cfg.SearchEngineAddr)
	}
}

func TestLoadCustomEnv(t *testing.T) {
	os.Clearenv()
	t.Setenv("S3_ENDPOINT", "garage:3900")
	t.Setenv("S3_ACCESS_KEY", "customkey")
	t.Setenv("S3_SECRET_KEY", "customsecret")
	t.Setenv("S3_BUCKET_NAME", "my-avatars")
	t.Setenv("S3_REGION", "garage")
	t.Setenv("S3_USE_SSL", "true")

	cfg := config.Load()
	if cfg.S3.Endpoint != "garage:3900" {
		t.Errorf("expected S3 endpoint garage:3900, got %s", cfg.S3.Endpoint)
	}
	if cfg.S3.AccessKey != "customkey" {
		t.Errorf("expected AccessKey customkey, got %s", cfg.S3.AccessKey)
	}
	if cfg.S3.SecretKey != "customsecret" {
		t.Errorf("expected SecretKey customsecret, got %s", cfg.S3.SecretKey)
	}
	if cfg.S3.BucketName != "my-avatars" {
		t.Errorf("expected BucketName my-avatars, got %s", cfg.S3.BucketName)
	}
	if !cfg.S3.UseSSL {
		t.Errorf("expected UseSSL true, got false")
	}
}
