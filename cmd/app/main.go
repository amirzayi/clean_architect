// Package main runs a web server with that dependencies.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/bradfitz/gomemcache/memcache"
	chim "github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	_ "modernc.org/sqlite"

	"github.com/amirzayi/clean_architect/internal/delivery"
	"github.com/amirzayi/clean_architect/internal/repository"
	"github.com/amirzayi/clean_architect/internal/service"
	"github.com/amirzayi/clean_architect/pkg/auth"
	"github.com/amirzayi/clean_architect/pkg/bus"
	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/amirzayi/clean_architect/pkg/config"
	"github.com/amirzayi/clean_architect/pkg/hash"
	"github.com/amirzayi/clean_architect/pkg/interceptor"
	"github.com/amirzayi/clean_architect/pkg/logger"
	"github.com/amirzayi/clean_architect/pkg/server/grpcserver"
	"github.com/amirzayi/clean_architect/pkg/server/webserver"
	"github.com/amirzayi/rahjoo/middleware"
	"github.com/amirzayi/rahjoo/middleware/cors"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.json", "config file path, eg: -config=/path/to/file.json")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	if err = run(ctx, cfg); err != nil {
		slog.Error(err.Error())
		return
	}
}

func CacheDriver(driver, url, prefix string) cache.Driver {
	switch driver {
	case "redis":
		opt, err := redis.ParseURL(url)
		if err != nil {
			panic("invalid redis server url")
			// return nil, err
		}
		client := redis.NewClient(opt)
		return cache.NewRedisDriver(client, prefix)
	case "memcached":
		mc := memcache.New(url)
		return cache.NewMemCachedDriver(mc, prefix)
	default:
		return cache.NewInMemoryDriver()
	}
}

func EventDriver(driver, url string, queues []string) (bus.Driver, error) {
	switch driver {
	case "redis":
		opt, err := redis.ParseURL(url)
		if err != nil {
			return nil, err
		}
		client := redis.NewClient(opt)
		return bus.NewRedisBroker(client), nil
	case "nats":
		return bus.NewNatsBroker(url)
	case "rabbitmq":
		return bus.NewRabbitBroker(url, queues)
	default:
		return bus.NewInMemoryDriver(queues), nil
	}
}

func run(ctx context.Context, cfg config.AppConfig) error {

	eventDriver, err := EventDriver(
		cfg.Event.Driver(),
		cfg.Event.ConnectionString(),
		[]string{}, // todo: add some queues
	)
	if err != nil {
		return err
	}

	cacheDriver := CacheDriver(
		cfg.Cache.Driver(),
		cfg.Cache.ConnectionString(),
		cfg.Cache.Prefix(),
	)
	if err = cacheDriver.Ping(ctx); err != nil {
		return err
	}

	db, err := sqlx.Connect(cfg.DB.Driver(), cfg.DB.ConnectionString())
	if err != nil {
		return fmt.Errorf("failed to connect database: %w", err)
	}

	driver, err := sqlite.WithInstance(db.DB, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to load database driver: %v", err)
	}
	migrator, err := migrate.NewWithDatabaseInstance("file://infra/migrations", "sqlite", driver)
	if err != nil {
		return fmt.Errorf("failed to setup migrator: %v", err)
	}
	if err = migrator.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to do migrate: %v", err)
	}

	var logWriters []io.Writer
	if cfg.Logger.Console() {
		logWriters = append(logWriters, os.Stdout)
	}
	if cfg.Logger.Directory() != "" {
		fileLogger := logger.NewFileLogger(logger.FileLoggerType(cfg.Logger.FileCreationMode()), cfg.Logger.Directory())
		logWriters = append(logWriters, fileLogger)
	}
	if cfg.Logger.RemoteURL() != "" {
		remoteLogger := logger.NewRemoteLogger(cfg.Logger.RemoteURL())
		logWriters = append(logWriters, remoteLogger)
	}

	logWriter := io.MultiWriter(logWriters...)
	defaultLogger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{AddSource: true, Level: slog.Level(cfg.Logger.Level())}))
	// set as global logger, no need to pass logger to another part of application
	slog.SetDefault(defaultLogger)

	webServerLogFile := logger.NewFileLogger(logger.FileLogHourly, "weblog")
	webServerLogWriter := io.MultiWriter(os.Stdout, webServerLogFile)
	webServerLogger := slog.NewLogLogger(slog.NewJSONHandler(webServerLogWriter, nil), slog.LevelInfo)

	// todo: configurable log writer(ex: ELK, prometheus, web-service, etc.)
	// specific logger used for server metric
	serverMetricLogger := slog.NewLogLogger(slog.NewJSONHandler(os.Stdout, nil), slog.LevelInfo)
	// specific logger used for server(grpc&http) panic
	serverPanicLogger := slog.NewLogLogger(slog.NewJSONHandler(os.Stdout, nil), slog.LevelInfo)

	repos := repository.NewSQLRepositories(db)

	authManager := auth.NewJWT(jwt.SigningMethodHS512, []byte(cfg.Auth.Secret()), cfg.Auth.LifeTime())

	services := service.NewServices(&service.Dependencies{
		Repositories: repos,
		Hasher:       hash.NewBcryptHasher(bcrypt.DefaultCost),
		AuthManager:  authManager,
		Cache:        cacheDriver,
		Event:        eventDriver,
		Logger:       defaultLogger,
	})

	gwMux := runtime.NewServeMux()

	muxHandler := http.NewServeMux()
	muxHandler.Handle("/", gwMux)

	apiHandler := middleware.Chain(muxHandler,
		cors.CORSHandler(),
		chim.Recoverer,
		middleware.EnforceJSON,
		chim.RealIP,
		chim.Logger,
	)

	webServer := webserver.New(apiHandler,
		webserver.WithAddress(cfg.Web.Address()),
		webserver.WithLogger(webServerLogger),
		webserver.WithTimeouts(
			cfg.Web.IdleTimeout(),
			cfg.Web.ReadTimeOut(),
			cfg.Web.WriteTimeout(),
			cfg.Web.ReadHeaderTimeout(),
			cfg.Web.ShutdownTimeout(),
		),
	)

	delivery.SetupHTTPRouter(muxHandler, webServerLogger, services, authManager)

	grpcServer := grpcserver.New(
		cfg.GRPC.Address(),
		cfg.GRPC.ShutdownTimeout(),
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxReceiveMsgSize()),
		grpc.ReadBufferSize(cfg.GRPC.ReadBufferSize()),
		grpc.ChainUnaryInterceptor(
			interceptor.ResponseTimeMeter(serverMetricLogger),
			interceptor.Recovery(serverPanicLogger),
		),
	)

	if cfg.GRPC.HasReflection() {
		reflection.Register(grpcServer)
	}

	// todo: configurable tls on grpc
	grpcDialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	delivery.SetupGRPC(grpcServer.Server, services)

	if err = delivery.SetupGRPCGateway(ctx, cfg.GRPC.Address(), gwMux, grpcDialOptions...); err != nil {
		return err
	}

	exitCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL)
	defer stop()

	errCh := make(chan error)

	go func() {
		if err = webServer.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to run web server: %w", err)
		}
	}()
	slog.Debug("web server initialized on " + cfg.Web.Address())

	go func() {
		if err = grpcServer.Run(); err != nil {
			errCh <- fmt.Errorf("failed to run grpc server: %w", err)
		}
	}()
	slog.Debug("grpc server initialized on " + cfg.GRPC.Address())

	select {
	case err = <-errCh:
		slog.Error(err.Error())

	case <-exitCtx.Done():
		slog.Debug("received terminate signal")
	}

	wg := sync.WaitGroup{}

	for _, f := range [...]func() error{
		webServer.GracefulShutdown,
		db.Close,
		cacheDriver.Close,
		eventDriver.Close,
		func() error { grpcServer.GracefulShutdown(); return nil },
	} {
		wg.Add(1)
		go func() {
			errCh <- f()
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(errCh)
	}()

	for shutdownError := range errCh {
		err = errors.Join(err, shutdownError)
	}
	return err
}
