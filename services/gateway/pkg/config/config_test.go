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
	if cfg.MCP.MaxSessionsPerIP != 100 {
		t.Errorf("expected default MCP.MaxSessionsPerIP 100, got %d", cfg.MCP.MaxSessionsPerIP)
	}
	if cfg.MCP.MaxSessions != 200 {
		t.Errorf("expected default MCP.MaxSessions 200, got %d", cfg.MCP.MaxSessions)
	}
	if cfg.MCP.CallsPerMin != 60 {
		t.Errorf("expected default MCP.CallsPerMin 60, got %d", cfg.MCP.CallsPerMin)
	}
}

func TestLoadMCPOptions(t *testing.T) {
	os.Clearenv()
	t.Setenv("MCP_MAX_SESSIONS_PER_IP", "50")
	t.Setenv("MCP_MAX_SESSIONS", "150")
	t.Setenv("MCP_CALLS_PER_MIN", "30")

	cfg := config.Load()
	if cfg.MCP.MaxSessionsPerIP != 50 {
		t.Errorf("expected MCP.MaxSessionsPerIP 50, got %d", cfg.MCP.MaxSessionsPerIP)
	}
	if cfg.MCP.MaxSessions != 150 {
		t.Errorf("expected MCP.MaxSessions 150, got %d", cfg.MCP.MaxSessions)
	}
	if cfg.MCP.CallsPerMin != 30 {
		t.Errorf("expected MCP.CallsPerMin 30, got %d", cfg.MCP.CallsPerMin)
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

func TestLoadLLMConfigDefaults(t *testing.T) {
	os.Clearenv()

	cfg := config.Load()
	if cfg.LLM.Provider != "stub" {
		t.Errorf("expected LLM.Provider stub, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.OpenAI.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default OpenAI BaseURL, got %s", cfg.LLM.OpenAI.BaseURL)
	}
	if cfg.LLM.OpenAI.ModelName != "gpt-4o-mini" {
		t.Errorf("expected default OpenAI ModelName gpt-4o-mini, got %s", cfg.LLM.OpenAI.ModelName)
	}
	if cfg.LLM.GigaChat.Scope != "GIGACHAT_API_PERS" {
		t.Errorf("expected default GigaChat Scope GIGACHAT_API_PERS, got %s", cfg.LLM.GigaChat.Scope)
	}
	if cfg.LLM.GigaChat.OAuthURL != "https://ngw.devices.sberbank.ru:9443/api/v2/oauth" {
		t.Errorf("expected default GigaChat OAuthURL, got %s", cfg.LLM.GigaChat.OAuthURL)
	}
	if cfg.LLM.GigaChat.BaseURL != "https://api.giga.chat/v1" {
		t.Errorf("expected default GigaChat BaseURL, got %s", cfg.LLM.GigaChat.BaseURL)
	}
	if cfg.LLM.GigaChat.Model != "GigaChat" {
		t.Errorf("expected default GigaChat Model GigaChat, got %s", cfg.LLM.GigaChat.Model)
	}
	if cfg.LLM.Temperature != 0.2 {
		t.Errorf("expected default LLM Temperature 0.2, got %f", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 2048 {
		t.Errorf("expected default LLM MaxTokens 2048, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.MaxConcurrent != 8 {
		t.Errorf("expected default LLM MaxConcurrent 8, got %d", cfg.LLM.MaxConcurrent)
	}
	if cfg.LLM.SystemPrompt == "" {
		t.Errorf("expected non-empty default SystemPrompt")
	}
}

func TestLoadLLMConfigCustom(t *testing.T) {
	os.Clearenv()
	t.Setenv("LLM_PROVIDER", "openai")
	t.Setenv("OPENAI_BASE_URL", "https://custom-openai.com/v1")
	t.Setenv("OPENAI_API_KEY", "sk-testkey")
	t.Setenv("OPENAI_MODEL_NAME", "gpt-4o")
	t.Setenv("GIGACHAT_CLIENT_ID", "gc-client")
	t.Setenv("GIGACHAT_CLIENT_SECRET", "gc-secret")
	t.Setenv("GIGACHAT_SCOPE", "GIGACHAT_API_CORP")
	t.Setenv("GIGACHAT_OAUTH_URL", "https://custom-oauth.ru/oauth")
	t.Setenv("GIGACHAT_BASE_URL", "https://custom-gigachat.ru/v1")
	t.Setenv("GIGACHAT_MODEL", "GigaChat-Pro")
	t.Setenv("GIGACHAT_CA_CERT", "/path/to/cert.pem")
	t.Setenv("LLM_TEMPERATURE", "0.7")
	t.Setenv("LLM_MAX_TOKENS", "4096")
	t.Setenv("LLM_MAX_CONCURRENT", "16")
	t.Setenv("RAG_SYSTEM_PROMPT", "Custom prompt")

	cfg := config.Load()
	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected LLM.Provider openai, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.OpenAI.BaseURL != "https://custom-openai.com/v1" {
		t.Errorf("expected custom OpenAI BaseURL, got %s", cfg.LLM.OpenAI.BaseURL)
	}
	if cfg.LLM.OpenAI.APIKey != "sk-testkey" {
		t.Errorf("expected custom OpenAI APIKey sk-testkey, got %s", cfg.LLM.OpenAI.APIKey)
	}
	if cfg.LLM.OpenAI.ModelName != "gpt-4o" {
		t.Errorf("expected custom OpenAI ModelName gpt-4o, got %s", cfg.LLM.OpenAI.ModelName)
	}
	if cfg.LLM.GigaChat.ClientID != "gc-client" {
		t.Errorf("expected custom GigaChat ClientID gc-client, got %s", cfg.LLM.GigaChat.ClientID)
	}
	if cfg.LLM.GigaChat.ClientSecret != "gc-secret" {
		t.Errorf("expected custom GigaChat ClientSecret gc-secret, got %s", cfg.LLM.GigaChat.ClientSecret)
	}
	if cfg.LLM.GigaChat.Scope != "GIGACHAT_API_CORP" {
		t.Errorf("expected custom GigaChat Scope GIGACHAT_API_CORP, got %s", cfg.LLM.GigaChat.Scope)
	}
	if cfg.LLM.GigaChat.OAuthURL != "https://custom-oauth.ru/oauth" {
		t.Errorf("expected custom GigaChat OAuthURL, got %s", cfg.LLM.GigaChat.OAuthURL)
	}
	if cfg.LLM.GigaChat.BaseURL != "https://custom-gigachat.ru/v1" {
		t.Errorf("expected custom GigaChat BaseURL, got %s", cfg.LLM.GigaChat.BaseURL)
	}
	if cfg.LLM.GigaChat.Model != "GigaChat-Pro" {
		t.Errorf("expected custom GigaChat Model GigaChat-Pro, got %s", cfg.LLM.GigaChat.Model)
	}
	if cfg.LLM.GigaChat.CACert != "/path/to/cert.pem" {
		t.Errorf("expected custom GigaChat CACert /path/to/cert.pem, got %s", cfg.LLM.GigaChat.CACert)
	}
	if cfg.LLM.Temperature != 0.7 {
		t.Errorf("expected custom LLM Temperature 0.7, got %f", cfg.LLM.Temperature)
	}
	if cfg.LLM.MaxTokens != 4096 {
		t.Errorf("expected custom LLM MaxTokens 4096, got %d", cfg.LLM.MaxTokens)
	}
	if cfg.LLM.MaxConcurrent != 16 {
		t.Errorf("expected custom LLM MaxConcurrent 16, got %d", cfg.LLM.MaxConcurrent)
	}
	if cfg.LLM.SystemPrompt != "Custom prompt" {
		t.Errorf("expected custom SystemPrompt, got %s", cfg.LLM.SystemPrompt)
	}
}
