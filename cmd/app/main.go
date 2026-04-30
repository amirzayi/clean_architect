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

	chim "github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-jwt/jwt/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	_ "modernc.org/sqlite"

	"github.com/amirzayi/clean_architect/internal/delivery"
	"github.com/amirzayi/clean_architect/internal/repository"
	"github.com/amirzayi/clean_architect/internal/service"
	"github.com/amirzayi/clean_architect/pkg/auth"
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

func run(ctx context.Context, cfg config.AppConfig) error {
	deps := dependencies{}
	sched, err := JobSchedulerDriver(cfg.Scheduler.Driver(), cfg.Scheduler.ConnectionString(), &deps)
	if err != nil {
		return err
	}
	go func() {
		for err = range sched.Start(ctx) {
			slog.Error(err.Error())
		}
	}()

	eventDriver, err := EventDriver(
		cfg.Event.Driver(),
		cfg.Event.ConnectionString(),
		[]string{}, // todo: add some queues
		&deps,
	)
	if err != nil {
		return err
	}

	cacheDriver, err := CacheDriver(
		cfg.Cache.Driver(),
		cfg.Cache.ConnectionString(),
		cfg.Cache.Prefix(),
		&deps,
	)
	if err != nil {
		return err
	}

	db, err := deps.getDBAndDoMigrate(cfg.DB.Driver(), cfg.DB.ConnectionString())
	if err != nil {
		return err
	}

	logWriter := logWriter(cfg.Logger)
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
		Scheduler:    sched,
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
	slog.Debug("web server initialized", "address", cfg.Web.Address())

	go func() {
		if err = grpcServer.Run(); err != nil {
			errCh <- fmt.Errorf("failed to run grpc server: %w", err)
		}
	}()
	slog.Debug("grpc server initialized", "address", cfg.GRPC.Address())

	select {
	case err = <-errCh:
		slog.Error(err.Error())

	case <-exitCtx.Done():
		slog.Debug("received terminate signal")
	}

	var wg sync.WaitGroup

	for _, f := range [...]func() error{
		webServer.GracefulShutdown,
		db.Close,
		deps.Close,
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
