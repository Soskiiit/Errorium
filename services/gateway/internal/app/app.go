package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/soskiiit/errorium/services/gateway/api"
	"github.com/soskiiit/errorium/services/gateway/domain"
	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/llm"
	"github.com/soskiiit/errorium/services/gateway/internal/mcp"
	"github.com/soskiiit/errorium/services/gateway/internal/metrics"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
	"github.com/soskiiit/errorium/services/gateway/internal/rag"
	"github.com/soskiiit/errorium/services/gateway/internal/repository"
	"github.com/soskiiit/errorium/services/gateway/internal/search"
	"github.com/soskiiit/errorium/services/gateway/internal/sso"
	"github.com/soskiiit/errorium/services/gateway/pkg/config"
	"github.com/soskiiit/errorium/services/gateway/pkg/db"
	jwtpkg "github.com/soskiiit/errorium/services/gateway/pkg/jwt"
	"github.com/soskiiit/errorium/services/gateway/pkg/mailer"
	"github.com/soskiiit/errorium/services/gateway/pkg/storage"
)

type App struct {
	cfg        *config.Config
	server     *http.Server
	database   *db.DB
	searchConn *grpc.ClientConn
}

func New(cfg *config.Config) *App {
	mux := http.NewServeMux()

	healthHandler := handler.NewHealthHandler()
	healthHandler.RegisterRoutes(mux)

	docsHandler := handler.NewDocsHandler(api.OpenAPIYAML)
	docsHandler.RegisterRoutes(mux)

	var searchClient search.SearchClient
	var graphClient graph.GraphClient
	var searchConn *grpc.ClientConn
	var coreChecker handler.CoreChecker

	if cfg.SearchEngineAddr != "" {
		conn, err := grpc.NewClient(cfg.SearchEngineAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("Failed to create gRPC client for search engine at %s: %v, falling back to stubs", cfg.SearchEngineAddr, err)
			searchClient = search.NewStub()
			graphClient = graph.NewStub()
		} else {
			searchConn = conn
			grpcSearch := search.NewGRPCClient(conn)
			searchClient = grpcSearch
			graphClient = graph.NewGRPCClient(conn)
			coreChecker = grpcSearch
		}
	} else {
		log.Println("SEARCH_ENGINE_ADDR is empty; using search and graph stubs (dev mode)")
		searchClient = search.NewStub()
		graphClient = graph.NewStub()
	}

	var database *db.DB
	var ragConfigRepo repository.RagConfigStore
	if cfg.DB.Host != "" {
		dbPool, err := db.New(cfg.DB)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
		} else {
			database = dbPool
			ragConfigRepo = repository.NewRagConfigRepository(database)
		}
	}

	openAIClient := llm.NewOpenAICompat(cfg.LLM.OpenAI.BaseURL, cfg.LLM.OpenAI.APIKey)
	var gigaChatClient llm.Client
	if cfg.LLM.GigaChat.CACert != "" {
		gc, err := llm.NewGigaChat(llm.GigaChatConfig{
			ClientID:     cfg.LLM.GigaChat.ClientID,
			ClientSecret: cfg.LLM.GigaChat.ClientSecret,
			Scope:        cfg.LLM.GigaChat.Scope,
			OAuthURL:     cfg.LLM.GigaChat.OAuthURL,
			BaseURL:      cfg.LLM.GigaChat.BaseURL,
			Model:        cfg.LLM.GigaChat.Model,
			CACert:       cfg.LLM.GigaChat.CACert,
		})
		if err != nil {
			log.Printf("GigaChat provider initialization failed: %v", err)
		} else {
			gigaChatClient = gc
		}
	} else if cfg.LLM.GigaChat.ClientID != "" {
		gc, _ := llm.NewGigaChatWithClient(llm.GigaChatConfig{
			ClientID:     cfg.LLM.GigaChat.ClientID,
			ClientSecret: cfg.LLM.GigaChat.ClientSecret,
			Scope:        cfg.LLM.GigaChat.Scope,
			OAuthURL:     cfg.LLM.GigaChat.OAuthURL,
			BaseURL:      cfg.LLM.GigaChat.BaseURL,
			Model:        cfg.LLM.GigaChat.Model,
		}, nil)
		gigaChatClient = gc
	}

	llmSem := rag.NewSemaphore(cfg.LLM.MaxConcurrent)
	ragOrch := rag.NewOrchestrator(
		ragConfigRepo,
		llmSem,
		openAIClient,
		gigaChatClient,
		cfg.LLM.OpenAI.APIKey,
		cfg.LLM.OpenAI.BaseURL,
		cfg.LLM.SystemPrompt,
		nil,
	)

	metricsInst := metrics.New()

	searchHandler := handler.NewSearchHandler(searchClient, graphClient, ragOrch).WithMetrics(metricsInst)
	searchHandler.RegisterRoutes(mux)

	tagsHandler := handler.NewTagsHandler()
	tagsHandler.RegisterRoutes(mux)

	statusHandler := handler.NewStatusHandler(cfg.SearchEngineAddr, coreChecker).WithRagConfigStore(ragConfigRepo)
	statusHandler.RegisterRoutes(mux)

	mcpServer := mcp.NewServer(searchClient, graphClient, ragOrch)
	mcpLimiter := mcp.NewLimiter(mcp.LimiterConfig{
		MaxSessionsPerIP: cfg.MCP.MaxSessionsPerIP,
		MaxSessions:      cfg.MCP.MaxSessions,
		CallsPerMin:      cfg.MCP.CallsPerMin,
	})
	mcpHandler := handler.NewMCPHandler(mcpServer, mcpLimiter)
	mcpHandler.RegisterRoutes(mux)

	if database != nil {
		userRepo := repository.NewUserRepository(database)
		sessionRepo := repository.NewSessionRepository(database)
		chatRepo := repository.NewChatRepository(database)
		tagRepo := repository.NewTagRepository(database)

		jwtMgr := jwtpkg.NewManager(cfg.JWT)
		authMW := middleware.Auth(jwtMgr)

		authHandler := handler.NewAuthHandler(userRepo, sessionRepo, jwtMgr, cfg.JWT.RefreshTokenExpireDays)
		authHandler.RegisterRoutes(mux)

		chatHandler := handler.NewChatHandler(chatRepo, tagRepo, searchClient, ragOrch)
		chatHandler.RegisterRoutes(mux, authMW)

		var avatarStore storage.AvatarStore
		if cfg.S3.Endpoint != "" {
			s3Store, err := storage.NewS3Storage(cfg.S3)
			if err != nil {
				log.Printf("Failed to initialize S3 storage: %v", err)
				avatarStore = storage.NewMemoryStore()
			} else {
				avatarStore = s3Store
			}
		} else {
			avatarStore = storage.NewMemoryStore()
		}

		userHandler := handler.NewUserHandler(userRepo, avatarStore)
		userHandler.RegisterRoutes(mux, authMW)

		passwordResetRepo := repository.NewPasswordResetRepository(database)
		mailerInstance := mailer.New(cfg.SMTP)
		passwordResetHandler := handler.NewPasswordResetHandler(
			userRepo,
			passwordResetRepo,
			sessionRepo,
			mailerInstance,
			cfg.FrontendURL,
		)
		passwordResetHandler.RegisterRoutes(mux, authMW)

		ssoStateRepo := repository.NewSsoStateRepository(database)
		ssoAccountRepo := repository.NewSSOAccountRepository(database)

		ssoRegistry := sso.NewRegistry()
		ssoRegistry.Register(sso.NewGitHubProvider(sso.ProviderConfig{
			ClientID:     cfg.SSO.GitHub.ClientID,
			ClientSecret: cfg.SSO.GitHub.ClientSecret,
		}))
		ssoRegistry.Register(sso.NewGoogleProvider(sso.ProviderConfig{
			ClientID:     cfg.SSO.Google.ClientID,
			ClientSecret: cfg.SSO.Google.ClientSecret,
		}))
		ssoRegistry.Register(sso.NewYandexProvider(sso.ProviderConfig{
			ClientID:     cfg.SSO.Yandex.ClientID,
			ClientSecret: cfg.SSO.Yandex.ClientSecret,
		}))

		ssoHandler := handler.NewSSOHandler(
			ssoRegistry,
			ssoStateRepo,
			ssoAccountRepo,
			userRepo,
			sessionRepo,
			jwtMgr,
			cfg.SSO.RedirectBase,
			cfg.FrontendURL,
			cfg.JWT.RefreshTokenExpireDays,
		)
		ssoHandler.RegisterRoutes(mux)

		auditRepo := repository.NewAuditRepository(database)

		if cfg.FirstAdminEmail != "" && cfg.FirstAdminPassword != "" {
			if err := BootstrapFirstAdmin(context.Background(), userRepo, cfg.FirstAdminEmail, cfg.FirstAdminPassword); err != nil {
				log.Printf("Bootstrap admin failed: %v", err)
			}
		}

		var coreAdminClient handler.CoreAdminClient
		if grpcSearch, ok := searchClient.(handler.CoreAdminClient); ok {
			coreAdminClient = grpcSearch
		}

		adminHandler := handler.NewAdminHandler(
			coreAdminClient,
			metricsInst,
			userRepo,
			ragConfigRepo,
			auditRepo,
		)
		adminMW := func(next http.Handler) http.Handler {
			return authMW(middleware.RequireAdmin(next))
		}
		adminHandler.RegisterRoutes(mux, adminMW)
	}

	corsMW := middleware.CORS()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           middleware.RequestLogger(corsMW(mux)),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      0, // WriteTimeout = 0 to allow long-running SSE streams!
		IdleTimeout:       60 * time.Second,
	}

	return &App{
		cfg:        cfg,
		server:     server,
		database:   database,
		searchConn: searchConn,
	}
}

// Handler returns the application root HTTP handler (wrapped with request logging and CORS).
func (a *App) Handler() http.Handler {
	return a.server.Handler
}

// Server returns the underlying http.Server instance.
func (a *App) Server() *http.Server {
	return a.server
}

func (a *App) Run() error {
	log.Printf("Gateway starting on :%s", a.cfg.Port)
	if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context) error {
	log.Println("Shutting down gateway...")
	err := a.server.Shutdown(ctx)
	if a.database != nil {
		if dbErr := a.database.Close(); dbErr != nil && err == nil {
			err = dbErr
		}
	}
	if a.searchConn != nil {
		if connErr := a.searchConn.Close(); connErr != nil && err == nil {
			err = connErr
		}
	}
	return err
}

// BootstrapFirstAdmin creates the initial administrator user if not already present.
// The operation is strictly idempotent and does not overwrite existing users.
func BootstrapFirstAdmin(ctx context.Context, userRepo repository.UserStore, email, password string) error {
	email = strings.TrimSpace(email)
	if email == "" || password == "" {
		return nil
	}

	existing, err := userRepo.GetByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil
	}

	parts := strings.Split(email, "@")
	username := parts[0]
	if username == "" {
		username = "admin"
	}

	pwHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("failed to hash first admin password: %w", err)
	}
	hashStr := string(pwHash)

	now := time.Now().UTC()
	adminUser := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: &hashStr,
		Role:         domain.RoleAdmin,
		Profession:   "Administrator",
		Gender:       "Not specified",
		Language:     "English",
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		return fmt.Errorf("failed to create first admin user: %w", err)
	}
	log.Printf("First administrator successfully bootstrapped: %s", email)
	return nil
}
