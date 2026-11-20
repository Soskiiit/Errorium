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
	if cfg.FrontendURL != "http://localhost:3000" {
		t.Errorf("expected FrontendURL http://localhost:3000, got %s", cfg.FrontendURL)
	}
	if cfg.SMTP.Port != 587 {
		t.Errorf("expected SMTP port 587, got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.From != "no-reply@errorium.ru" {
		t.Errorf("expected SMTP from no-reply@errorium.ru, got %s", cfg.SMTP.From)
	}
	if cfg.SMTP.Host != "" {
		t.Errorf("expected empty default SMTP host, got %s", cfg.SMTP.Host)
	}
	if cfg.SSO.RedirectBase != "http://localhost:8000" {
		t.Errorf("expected SSO.RedirectBase http://localhost:8000, got %s", cfg.SSO.RedirectBase)
	}
	if cfg.SSO.GitHub.ClientID != "" || cfg.SSO.Google.ClientID != "" || cfg.SSO.Yandex.ClientID != "" {
		t.Errorf("expected empty default SSO client IDs")
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

func TestLoadSMTPAndFrontendConfig(t *testing.T) {
	os.Clearenv()
	t.Setenv("FRONTEND_URL", "https://app.errorium.ru")
	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_PORT", "465")
	t.Setenv("SMTP_USER", "smtpuser")
	t.Setenv("SMTP_PASSWORD", "smtppass")
	t.Setenv("SMTP_FROM", "support@errorium.ru")

	cfg := config.Load()
	if cfg.FrontendURL != "https://app.errorium.ru" {
		t.Errorf("expected FrontendURL https://app.errorium.ru, got %s", cfg.FrontendURL)
	}
	if cfg.SMTP.Host != "smtp.example.com" {
		t.Errorf("expected SMTP host smtp.example.com, got %s", cfg.SMTP.Host)
	}
	if cfg.SMTP.Port != 465 {
		t.Errorf("expected SMTP port 465, got %d", cfg.SMTP.Port)
	}
	if cfg.SMTP.User != "smtpuser" {
		t.Errorf("expected SMTP user smtpuser, got %s", cfg.SMTP.User)
	}
	if cfg.SMTP.Password != "smtppass" {
		t.Errorf("expected SMTP password smtppass, got %s", cfg.SMTP.Password)
	}
	if cfg.SMTP.From != "support@errorium.ru" {
		t.Errorf("expected SMTP from support@errorium.ru, got %s", cfg.SMTP.From)
	}
}

func TestLoadSSOConfig(t *testing.T) {
	os.Clearenv()
	t.Setenv("SSO_REDIRECT_BASE", "https://api.errorium.ru")
	t.Setenv("GITHUB_CLIENT_ID", "gh-id")
	t.Setenv("GITHUB_CLIENT_SECRET", "gh-secret")
	t.Setenv("GOOGLE_CLIENT_ID", "goog-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "goog-secret")
	t.Setenv("YANDEX_CLIENT_ID", "ya-id")
	t.Setenv("YANDEX_CLIENT_SECRET", "ya-secret")

	cfg := config.Load()
	if cfg.SSO.RedirectBase != "https://api.errorium.ru" {
		t.Errorf("expected SSO redirect base https://api.errorium.ru, got %s", cfg.SSO.RedirectBase)
	}
	if cfg.SSO.GitHub.ClientID != "gh-id" || cfg.SSO.GitHub.ClientSecret != "gh-secret" {
		t.Errorf("unexpected GitHub config: %+v", cfg.SSO.GitHub)
	}
	if cfg.SSO.Google.ClientID != "goog-id" || cfg.SSO.Google.ClientSecret != "goog-secret" {
		t.Errorf("unexpected Google config: %+v", cfg.SSO.Google)
	}
	if cfg.SSO.Yandex.ClientID != "ya-id" || cfg.SSO.Yandex.ClientSecret != "ya-secret" {
		t.Errorf("unexpected Yandex config: %+v", cfg.SSO.Yandex)
	}
}
