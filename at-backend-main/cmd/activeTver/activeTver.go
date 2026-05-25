package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"activeTver/internal/delivery/http/v1/middleware"
	"activeTver/internal/infrastructure/db/postgres"
	"activeTver/internal/infrastructure/external/cache"
	"activeTver/internal/infrastructure/external/email"
	"activeTver/internal/infrastructure/external/file_manager"
	template "activeTver/internal/infrastructure/external/template_renderer"
	"activeTver/internal/pkg/config"
	"activeTver/internal/pkg/logs"
	"activeTver/internal/pkg/pgclient"
	"activeTver/internal/pkg/raycast"
	"activeTver/internal/pkg/tokens"
	"activeTver/internal/usecase"

	"activeTver/internal/delivery/http/v1/handlers"
)

const (
	op            = "cmd.activeTver"
	correlationID = "activeTver"
)

func main() {
	logs.Setup()
	defer logs.Flush()

	logs.Info(
		"App starting",
		correlationID,
		op,
	)

	cfg, err := config.GetConfig()
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	// init db connection
	dbClientCfg := pgclient.Config{
		Username: cfg.Storage.User,
		Password: cfg.Storage.Password,
		Host:     cfg.Storage.Host,
		Port:     "5432",
		Database: cfg.Storage.Name,
	}

	pool, err := pgclient.NewClient(context.Background(), 5, dbClientCfg)
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID, op,
		)

		return
	}
	defer pool.Close()

	transactor := postgres.NewTransactor(pool)

	tokensManager, err := tokens.NewTokenManager(
		[]byte(cfg.Tokens.JWTSecretKey),
		cfg.Tokens.JWTDuration,
		128,
	)
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	s3, err := filemanager.NewS3Client(
		context.Background(),
		cfg.ObjectStorage.AccessKeyID,
		cfg.ObjectStorage.AccessKeySecret,
		cfg.ObjectStorage.SessionToken,
		cfg.ObjectStorage.Bucket,
		cfg.ObjectStorage.Endpoint,
		cfg.ObjectStorage.AccessHost,
	)
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	emailClient, err := email.NewClient(cfg.Email.Host, cfg.Email.Port, cfg.Email.SenderEmail, cfg.Email.SenderPassword)
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	templateRender, err := template.NewTemplateRenderer("./resources/")
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	mailCodesCache := cache.NewCacheStorage[string, *usecase.MailCode](time.Minute * 5)

	// init storage
	claimsRepository := postgres.NewClaimsRepository(pool)

	userRepository := postgres.NewUserRepository(pool)

	tokensRepository := postgres.NewRefreshTokensRepository(pool)

	adminRepository := postgres.NewAdminRepository(pool)

	announcementRepository := postgres.NewAnnouncementRepository(pool)

	categoriesRepository := postgres.NewCategoriesRepository(pool)

	tverCentral, err := raycast.LoadPolygonFromFile("./resources/polygons/tver.central")
	if err != nil {
		logs.Error(
			err.Error(),
			correlationID,
			op,
		)

		return
	}

	categoriesService := usecase.CategoriesService{
		Repository: categoriesRepository,
	}

	mapService := usecase.MapService{
		Districts: map[string]raycast.Polygon{
			"tver.central": tverCentral,
		},
	}

	claimsService := usecase.ClaimsService{
		Repository:                 claimsRepository,
		FileManager:                s3,
		MaxPageSize:                100,
		CategoriesService:          &categoriesService,
		MapService:                 mapService,
		CompletedClaimsNoOlderThan: time.Hour * 24 * 14,
	}

	userService := usecase.UserService{
		Repository:  userRepository,
		MaxPageSize: 100,
	}

	adminService := usecase.AdminService{
		Repository: adminRepository,
	}

	authService := usecase.AuthService{
		TokensRepository: tokensRepository,
		UserService:      userService,
		TokenManager:     tokensManager,
		EmailClient:      emailClient,
		TemplateRenderer: templateRender,
		CacheStorage:     mailCodesCache,

		RefreshTokenExpirationTime: time.Hour * 24 * 30 * 3, // 3 month

		MailCodeExpirationTime: time.Minute * 5,
		MailCodeMaxAttempts:    5,
		MailCodeLen:            6,

		SupportEmail: cfg.Email.SupportEmail,

		Transactor: transactor,
	}

	announcementService := usecase.AnnouncementService{
		Repository:  announcementRepository,
		MaxPageSize: 100,
	}

	categoriesHandler := handlers.Categories{
		Service: &categoriesService,
	}

	authHandler := handlers.Auth{
		Service: authService,
	}

	claimsHandler := handlers.Claims{
		Service: claimsService,
	}

	announcementHandler := handlers.Announcement{
		Service: announcementService,
	}

	adminHandler := handlers.Admin{
		ClaimsService:       claimsService,
		AdminService:        adminService,
		AnnouncementService: announcementService,
		UserService:         userService,
		CategoriesService:   &categoriesService,
	}

	userHandler := handlers.User{
		Service: userService,
	}

	mapHandler := handlers.Map{
		ClaimsService: claimsService,
		MapService:    mapService,
	}

	authMiddleware := middleware.Auth(&tokensManager)

	mainMux := http.NewServeMux()

	authMux := authHandler.Setup(authMiddleware)

	claimsMux := claimsHandler.Setup()
	wrappedClaimsMux := middleware.CheckUserBlocked(claimsMux, userService.IsBlockedByUID)
	wrappedClaimsMux = authMiddleware(wrappedClaimsMux)

	adminMux := adminHandler.Setup()
	wrappedAdminMux := middleware.SetupAdminDataOnContext(adminMux, adminService.GetAdminDataByUID)
	//wrappedAdminMux = middleware.IsBlocked(wrappedAdminMux, userService.IsBlockedByID)
	wrappedAdminMux = authMiddleware(wrappedAdminMux)

	userMux := userHandler.Setup()
	wrappedUserMux := middleware.CheckUserBlocked(userMux, userService.IsBlockedByUID)
	wrappedUserMux = authMiddleware(wrappedUserMux)

	announcementMux := announcementHandler.Setup()
	wrappedAnnouncementMux := middleware.CheckUserBlocked(announcementMux, userService.IsBlockedByUID)
	wrappedAnnouncementMux = authMiddleware(wrappedAnnouncementMux)

	categoriesMux := categoriesHandler.Setup()
	wrappedCategoriesMux := middleware.CheckUserBlocked(categoriesMux, userService.IsBlockedByUID)
	wrappedCategoriesMux = authMiddleware(wrappedCategoriesMux)

	mapMux := mapHandler.Setup()
	wrappedMapMux := middleware.CheckUserBlocked(mapMux, userService.IsBlockedByUID)
	wrappedMapMux = authMiddleware(wrappedMapMux)

	mainMux.Handle("/v1/auth/", http.StripPrefix("/v1/auth", authMux))
	mainMux.Handle("/v1/my/claims/", http.StripPrefix("/v1/my/claims", wrappedClaimsMux))
	mainMux.Handle("/v1/me/", http.StripPrefix("/v1/me", wrappedUserMux))
	mainMux.Handle("/admin/v1/", http.StripPrefix("/admin/v1", wrappedAdminMux))
	mainMux.Handle("/v1/announcements/", http.StripPrefix("/v1/announcements", wrappedAnnouncementMux))
	mainMux.Handle("/v1/map/", http.StripPrefix("/v1/map", wrappedMapMux))
	mainMux.Handle("/v1/categories/", http.StripPrefix("/v1/categories", wrappedCategoriesMux))

	wrappedMainMux := middleware.Cors(mainMux)                     // 4
	wrappedMainMux = middleware.Logger(wrappedMainMux)             // 3
	wrappedMainMux = middleware.SetupCorrelationID(wrappedMainMux) // 2
	wrappedMainMux = middleware.SetupContextValues(wrappedMainMux) // 1

	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           wrappedMainMux,
		ReadTimeout:       cfg.Server.ReadTimeout,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
	}

	run(context.Background(), server)

	logs.Info(
		"App closed",
		correlationID,
		op,
	)
}

func run(ctx context.Context, server *http.Server) {
	var wg sync.WaitGroup
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	wg.Add(1)
	go func() {
		logs.Info(
			"HTTP server started",
			correlationID,
			op,
		)
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			logs.Info(
				"HTTP server closed",
				correlationID,
				op,
			)
			wg.Done()
			return
		}

		logs.Error(
			"HTTP server closed with error: "+err.Error(),
			correlationID,
			op,
		)
	}()

	wg.Add(1)
	go func() {
		<-ctx.Done()

		err := server.Shutdown(ctx)
		if err != nil {
			logs.Error(
				"HTTP server shutdown error: "+err.Error(),
				correlationID,
				op,
			)
		}

		wg.Done()
	}()

	wg.Wait()
}
