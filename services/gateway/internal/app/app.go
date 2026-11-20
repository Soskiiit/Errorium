package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/soskiiit/errorium/services/gateway/api"
	"github.com/soskiiit/errorium/services/gateway/internal/graph"
	"github.com/soskiiit/errorium/services/gateway/internal/handler"
	"github.com/soskiiit/errorium/services/gateway/internal/middleware"
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

	searchHandler := handler.NewSearchHandler(searchClient, graphClient)
	searchHandler.RegisterRoutes(mux)

	tagsHandler := handler.NewTagsHandler()
	tagsHandler.RegisterRoutes(mux)

	statusHandler := handler.NewStatusHandler(cfg.SearchEngineAddr, coreChecker)
	statusHandler.RegisterRoutes(mux)

	var database *db.DB
	if cfg.DB.Host != "" {
		dbPool, err := db.New(cfg.DB)
		if err != nil {
			log.Printf("Failed to connect to database: %v", err)
		} else {
			database = dbPool
			userRepo := repository.NewUserRepository(database)
			sessionRepo := repository.NewSessionRepository(database)
			chatRepo := repository.NewChatRepository(database)
			tagRepo := repository.NewTagRepository(database)

			jwtMgr := jwtpkg.NewManager(cfg.JWT)
			authMW := middleware.Auth(jwtMgr)

			authHandler := handler.NewAuthHandler(userRepo, sessionRepo, jwtMgr, cfg.JWT.RefreshTokenExpireDays)
			authHandler.RegisterRoutes(mux)

			chatHandler := handler.NewChatHandler(chatRepo, tagRepo, searchClient)
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
		}
	}

	corsMW := middleware.CORS()

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      corsMW(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		cfg:        cfg,
		server:     server,
		database:   database,
		searchConn: searchConn,
	}
}

// Handler returns the application root HTTP handler (wrapped with CORS).
func (a *App) Handler() http.Handler {
	return a.server.Handler
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
