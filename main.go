package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"tugas2/app/repository"
	"tugas2/app/service"
	"tugas2/config"
	"tugas2/database"
	"tugas2/helper"
	"tugas2/route"

	_ "tugas2/docs"
)

const minSecretLength = 32

// Blok komentar di bawah ini adalah informasi UMUM dokumentasi API
//
// @title API KRS Praktikum Backend Lanjut
// @version 1.0
// @description API pendaftaran kelas (KRS), ulasan kelas, dan notifikasi
// @description Seluruh endpoint selain /auth memerlukan access token
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Isi dengan: Bearer <access_token>
func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	role := flag.String("role", config.GetEnv("APP_ROLE", "all"), "api | worker | all")
	flag.Parse()

	runAPI := *role == "api" || *role == "all"
	runWorker := *role == "worker" || *role == "all"
	if !runAPI && !runWorker {
		logger.Error("role tidak dikenal", slog.String("role", *role))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var app *fiber.App
	if runAPI {
		app = startAPI(logger, pool)
		if app == nil {
			os.Exit(1)
		}
	}

	var stopWorker func()
	if runWorker {
		stopWorker = startWorker(ctx, logger, pool)
		if stopWorker == nil {
			os.Exit(1)
		}
	}

	logger.Info("aplikasi berjalan", slog.String("role", *role))

	<-ctx.Done()

	// HTTP ditutup lebih dulu, baru pekerja di belakangnya.
	if app != nil {
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			logger.Error("shutdown HTTP tidak rapi", slog.String("error", err.Error()))
		}
	}
	if stopWorker != nil {
		stopWorker()
	}

	logger.Info("aplikasi berhenti dengan rapi")
}

func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		return nil
	}

	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil
	}
	closeMongo := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

	studentRepository := repository.NewStudentRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	roleRepository := repository.NewRoleRepository(pool)
	courseRepository := repository.NewCourseRepository(pool)
	supervisorRepository := repository.NewSupervisorRepository(pool)
	outboxRepository := repository.NewOutboxRepository(pool)
	notificationRepository := repository.NewNotificationRepository(pool)
	txManager := repository.NewTxManager(pool)

	reviewRepository := repository.NewReviewRepository(mongoDB)
	if err := reviewRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index MongoDB", slog.String("error", err.Error()))
		closeMongo()
		return nil
	}

	portfolioRepository := repository.NewPortfolioRepository(mongoDB)
	if err := portfolioRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index portofolio", slog.String("error", err.Error()))
		closeMongo()
		return nil
	}

	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		closeMongo()
		return nil
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	// Inspector tidak menghubungi Redis saat dibuat, jadi API tetap bisa
	// menyala walaupun Redis mati.
	inspector := asynq.NewInspector(config.RedisOpt())

	app := config.NewApp(logger, route.Dependencies{
		Pool:          pool,
		Mongo:         mongoClient,
		EnableSwagger: config.GetEnv("SWAGGER_ENABLED", "false") == "true",
		JWT:           jwtManager,
		Permissions:   permissions,

		StudentService: service.NewStudentService(studentRepository, permissions),
		UserService:    service.NewUserService(userRepository, permissions),
		AuthService: service.NewAuthService(
			userRepository, tokenRepository, jwtManager, permissions,
			time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
		),
		CourseService:      service.NewCourseService(courseRepository, outboxRepository, txManager),
		SupervisionService: service.NewSupervisionService(supervisorRepository, txManager),
		ReviewService:      service.NewReviewService(reviewRepository, courseRepository, permissions),
		PortfolioService:   service.NewPortfolioService(portfolioRepository, studentRepository, permissions),

		NotificationService: service.NewNotificationService(
			notificationRepository, outboxRepository, inspector),
	}, config.GetEnv("ALLOWED_ORIGINS", ""))

	app.Hooks().OnShutdown(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = inspector.Close()
		return mongoClient.Disconnect(ctx)
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("HTTP berjalan", slog.String("port", port))
	return app
}
