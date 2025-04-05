// Package main runs a web server with that dependencies.
package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/amirzayi/rahjoo/middleware"
	chim "github.com/go-chi/chi/v5/middleware"
	_ "github.com/go-sql-driver/mysql"
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
	"github.com/amirzayi/clean_architect/pkg/config"
	"github.com/amirzayi/clean_architect/pkg/hash"
	"github.com/amirzayi/clean_architect/pkg/interceptor"
	"github.com/amirzayi/clean_architect/pkg/logger"
	"github.com/amirzayi/clean_architect/pkg/server/grpcserver"
	"github.com/amirzayi/clean_architect/pkg/server/webserver"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var configPath string
	flag.StringVar(&configPath, "config", "config.json", "config file path, eg: -config=/path/to/file.json")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfigOrDefault(configPath)
	if err != nil {
		return err
	}

	db, err := sql.Open(cfg.DB().Driver(), cfg.DB().ConnectionString())
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	var logWriters []io.Writer
	if cfg.Logger().Console() {
		logWriters = append(logWriters, os.Stdout)
	}
	if cfg.Logger().Directory() != "" {
		fileLogger := logger.NewFileLogger(logger.FileLoggerType(cfg.Logger().FileCreationMode()), cfg.Logger().Directory())
		logWriters = append(logWriters, fileLogger)
	}
	if cfg.Logger().RemoteURL() != "" {
		remoteLogger := logger.NewRemoteLogger(cfg.Logger().RemoteURL())
		logWriters = append(logWriters, remoteLogger)
	}

	logWriter := io.MultiWriter(logWriters...)
	defaultLogger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{AddSource: true, Level: slog.Level(cfg.Logger().Level())}))
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

	services := service.NewServices(&service.Dependencies{
		Repositories: repos,
		Hasher:       hash.NewBcryptHasher(bcrypt.DefaultCost),
	})

	gwMux := runtime.NewServeMux()

	muxHandler := http.NewServeMux()
	muxHandler.Handle("/", gwMux)

	apiHandler := middleware.Chain(muxHandler,
		chim.Recoverer,
		middleware.EnforceJSON,
		chim.RealIP,
		chim.Logger,
	)

	webServer := webserver.New(
		webserver.WithHandler(apiHandler),
		webserver.WithAddress(cfg.Web().Address()),
		webserver.WithLogger(webServerLogger),
		webserver.WithTimeouts(
			cfg.Web().IdleTimeout(),
			cfg.Web().ReadTimeOut(),
			cfg.Web().WriteTimeout(),
			cfg.Web().ReadHeaderTimeout(),
			cfg.Web().ShutdownTimeout(),
		),
	)

	delivery.SetupHTTPRouter(muxHandler, webServerLogger, services)

	grpcServer := grpcserver.New(
		cfg.GRPC().Address(),
		cfg.GRPC().ShutdownTimeout(),
		grpc.MaxRecvMsgSize(cfg.GRPC().MaxReceiveMsgSize()),
		grpc.ReadBufferSize(cfg.GRPC().ReadBufferSize()),
		grpc.ChainUnaryInterceptor(
			interceptor.ResponseTimeMeter(serverMetricLogger),
			interceptor.Recovery(serverPanicLogger),
		),
	)

	if cfg.GRPC().HasReflection() {
		reflection.Register(grpcServer.Server())
	}

	// todo: configurable tls on grpc
	grpcDialOptions := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	delivery.SetupGRPC(grpcServer.Server(), services)

	if err = delivery.SetupGRPCGateway(ctx, cfg.GRPC().Address(), gwMux, grpcDialOptions...); err != nil {
		return err
	}

	errCh := make(chan error, 4)

	wg := sync.WaitGroup{}
	wg.Add(5)

	go func() {
		defer wg.Done()
		if err = webServer.Run(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to run web server: %w", err)
		}
	}()
	log.Printf("web server initialized on %s", cfg.Web().Address())

	go func() {
		defer wg.Done()
		if err = grpcServer.Run(); err != nil {
			errCh <- fmt.Errorf("failed to run grpc server: %w", err)
		}
	}()
	log.Printf("grpc server initialized on %s", cfg.GRPC().Address())

	exitCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGKILL)

	select {
	case err = <-errCh:
		stop()

	case <-exitCtx.Done():
		log.Println("received terminate signal")
		go func() {
			defer wg.Done()
			grpcServer.GracefulShutdown()
		}()
		go func() {
			defer wg.Done()
			errCh <- webServer.GracefulShutdown()
		}()
		go func() {
			defer wg.Done()
			errCh <- db.Close()
		}()
	}

	wg.Wait()
	close(errCh)

	var errs error
	for err = range errCh {
		errs = errors.Join(err)
	}
	return errs
}
