package main

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
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

	redisError, natsError, memcacheError, rabbitError error
	redisOnce, natsOnce, memcacheOnce, rabbitmqOnce   sync.Once
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
	var err error
	for _, client := range []io.Closer{d.redisClient, d.memcacheClient, d.rabbitmqChannel, d.rabbitmqConnection} {
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

func JobSchedulerDriver(driver, url string, deps *dependencies) (scheduler.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		return scheduler.NewRedisScheduler(redisClient, time.Second, 1), err
	default:
		// return in memory driver instead of nil
		return nil, nil
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
