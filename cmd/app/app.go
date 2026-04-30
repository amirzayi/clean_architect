package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/amirzayi/clean_architect/pkg/bus"
	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/amirzayi/clean_architect/pkg/config"
	"github.com/amirzayi/clean_architect/pkg/logger"
	"github.com/amirzayi/clean_architect/pkg/scheduler"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type dependencies struct {
	redisClient        *redis.Client
	natsClient         *nats.Conn
	memcacheClient     *memcache.Client
	rabbitmqChannel    *amqp.Channel
	rabbitmqConnection *amqp.Connection
	db                 *sqlx.DB

	redisError, natsError, memcacheError, rabbitError, dbError error
	redisOnce, natsOnce, memcacheOnce, rabbitmqOnce, dbOnce    sync.Once
}

var ErrRedisConnectionTimeout = errors.New("redis connection timeout")

func (d *dependencies) getRedisClient(url string) (*redis.Client, error) {
	d.redisOnce.Do(func() {
		opt, err := redis.ParseURL(url)
		if err != nil {
			d.redisError = err
			return
		}
		redisClient := redis.NewClient(opt)
		ctx, cancel := context.WithTimeoutCause(context.Background(), 5*time.Second, ErrRedisConnectionTimeout)
		defer cancel()
		err = redisClient.Ping(ctx).Err()
		if err != nil {
			d.redisError = err
			return
		}
		d.redisClient = redisClient
	})
	return d.redisClient, d.redisError
}

func (d *dependencies) getMemcacheClient(url string) (*memcache.Client, error) {
	d.memcacheOnce.Do(func() {
		memcacheClient := memcache.New(url)
		if err := memcacheClient.Ping(); err != nil {
			d.memcacheError = err
			return
		}
		d.memcacheClient = memcacheClient
	})
	return d.memcacheClient, d.memcacheError
}

func (d *dependencies) getNatsClient(url string) (*nats.Conn, error) {
	d.natsOnce.Do(func() {
		c, err := nats.Connect(url)
		if err != nil {
			d.natsError = err
			return
		}
		d.natsClient = c
	})
	return d.natsClient, d.natsError
}

func (d *dependencies) getRabbitmqChannel(url string) (*amqp.Channel, error) {
	d.rabbitmqOnce.Do(func() {
		conn, err := amqp.Dial(url)
		if err != nil {
			d.rabbitError = err
			return
		}
		channel, err := conn.Channel()
		if err != nil {
			d.rabbitError = err
			return
		}
		d.rabbitmqConnection = conn
		d.rabbitmqChannel = channel
	})
	return d.rabbitmqChannel, d.rabbitError
}

func (d *dependencies) getDBAndDoMigrate(driver, url string) (*sqlx.DB, error) {
	d.dbOnce.Do(func() {
		db, err := sqlx.Connect(driver, url)
		if err != nil {
			d.dbError = fmt.Errorf("failed to connect database: %w", err)
			return
		}
		if err = db.Ping(); err != nil {
			d.dbError = fmt.Errorf("failed to ping database: %w", err)
			return
		}
		migratorDriver, err := dbMigratorDriver(driver, db.DB)
		if err != nil {
			d.dbError = fmt.Errorf("failed to load database migrator driver: %v", err)
			return
		}
		migrator, err := migrate.NewWithDatabaseInstance("file://infra/migrations", driver, migratorDriver)
		if err != nil {
			d.dbError = fmt.Errorf("failed to setup migrator: %v", err)
			return
		}
		if err = migrator.Up(); err != nil && err != migrate.ErrNoChange {
			d.dbError = fmt.Errorf("failed to do migrate: %v", err)
			return
		}
		d.db = db
	})
	return d.db, d.dbError
}

func dbMigratorDriver(driver string, db *sql.DB) (dbDriver database.Driver, err error) {
	switch driver {
	case "sqlite":
		dbDriver, err = sqlite.WithInstance(db, &sqlite.Config{})
	case "postgres":
		dbDriver, err = postgres.WithInstance(db, &postgres.Config{})
	case "mysql":
		dbDriver, err = mysql.WithInstance(db, &mysql.Config{})
	default:
		err = errors.New("undefined database migrator driver")
	}
	return
}

func (d *dependencies) Close() error {
	if d.natsClient != nil {
		d.natsClient.Close()
	}
	var clients []io.Closer
	if d.redisClient != nil {
		clients = append(clients, d.redisClient)
	}
	if d.memcacheClient != nil {
		clients = append(clients, d.memcacheClient)
	}
	if d.rabbitmqChannel != nil {
		clients = append(clients, d.rabbitmqChannel)
	}
	if d.rabbitmqConnection != nil {
		clients = append(clients, d.rabbitmqConnection)
	}
	var err error
	for _, client := range clients {
		if client != nil {
			err = errors.Join(err, client.Close())
		}
	}
	return err
}

func CacheDriver(driver, url, prefix string, deps *dependencies) (cache.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		return cache.NewRedisDriver(redisClient, prefix), err
	case "memcached":
		memcacheClient, err := deps.getMemcacheClient(url)
		return cache.NewMemCachedDriver(memcacheClient, prefix), err
	default:
		return cache.NewInMemoryDriver(), nil
	}
}

func EventDriver(driver, url string, queues []string, deps *dependencies) (bus.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		return bus.NewRedisBroker(redisClient), err
	case "nats":
		natsClient, err := deps.getNatsClient(url)
		return bus.NewNatsBroker(natsClient), err
	case "rabbitmq":
		rabbitChannel, err := deps.getRabbitmqChannel(url)
		if err != nil {
			return nil, err
		}
		return bus.NewRabbitBroker(rabbitChannel, queues)
	default:
		return bus.NewInMemoryDriver(queues), nil
	}
}

func JobSchedulerDriver(cfg config.SchedulerConfig, deps *dependencies) (scheduler.Driver, error) {
	switch cfg.Driver() {
	case "redis":
		redisClient, err := deps.getRedisClient(cfg.ConnectionString())
		return scheduler.NewRedisScheduler(redisClient, cfg.RunEvery(), cfg.Concurrency()), err
	case "sqlite":
		db, err := deps.getDBAndDoMigrate(cfg.Driver(), cfg.ConnectionString())
		return scheduler.NewSQLScheduler(db, cfg.RunEvery(), cfg.Concurrency()), err
	default:
		return scheduler.NewDiscard(), nil
	}
}

func logWriter(cfg config.LoggerConfig) io.Writer {
	var logWriters []io.Writer
	if cfg.Console() {
		logWriters = append(logWriters, os.Stdout)
	}
	if cfg.Directory() != "" {
		fileLogger := logger.NewFileLogger(logger.FileLoggerType(cfg.FileCreationMode()), cfg.Directory())
		logWriters = append(logWriters, fileLogger)
	}
	if cfg.RemoteURL() != "" {
		remoteLogger := logger.NewRemoteLogger(cfg.RemoteURL())
		logWriters = append(logWriters, remoteLogger)
	}
	if len(logWriters) == 0 {
		return io.Discard
	}
	return io.MultiWriter(logWriters...)
}
