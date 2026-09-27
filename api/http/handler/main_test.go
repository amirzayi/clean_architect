package handler_test

import (
	"io"
	"log"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"github.com/amirzayi/clean_architect/api/http/handler"
	"github.com/amirzayi/clean_architect/internal/domain"
	"github.com/amirzayi/clean_architect/internal/repository"
	"github.com/amirzayi/clean_architect/internal/service"
	"github.com/amirzayi/clean_architect/pkg/auth"
	"github.com/amirzayi/clean_architect/pkg/bus"
	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/amirzayi/clean_architect/pkg/hash"
)

var (
	mux        = http.NewServeMux()
	adminToken string
	userToken  string
)

func TestMain(m *testing.M) {
	db, err := sqlx.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		log.Fatalf("failed to load database driver: %v", err)
	}
	migrator, err := migrate.NewWithDatabaseInstance("file://../../../infra/migrations", "sqlite3", driver)
	if err != nil {
		log.Fatalf("failed to setup migrator: %v", err)
	}
	if err = migrator.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("failed to do migrate: %v", err)
	}

	repos := repository.NewSQLRepositories(db)

	authManager := auth.NewJWT(jwt.SigningMethodHS512, []byte("testing_key"), time.Hour)
	adminToken, err = authManager.CreateToken(uuid.New(), string(domain.UserRoleAdmin))
	if err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}
	userToken, err = authManager.CreateToken(uuid.New(), string(domain.UserRoleNormal))
	if err != nil {
		log.Fatalf("failed to generate token: %v", err)
	}

	services := service.NewServices(&service.Dependencies{
		Repositories: repos,
		Hasher:       hash.NewBcryptHasher(bcrypt.DefaultCost),
		AuthManager:  authManager,
		Cache:        cache.New(cache.NewInMemoryDriver()),
		Event:        bus.New(bus.NewInMemoryDriver([]string{})),
		Logger:       slog.Default(),
	})
	handler.Register(mux, log.New(io.Discard, "", 0), services, authManager)
	m.Run()
}
