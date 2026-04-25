package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/amirzayi/clean_architect/pkg/bus"
	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/amirzayi/clean_architect/pkg/scheduler"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/nats-io/nats.go"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
)

type dependencies struct {
	redisOnce   *sync.Once
	redisClient *redis.Client
	redisError  error

	natsOnce   *sync.Once
	natsClient *nats.Conn
	natsError  error

	memcacheOnce   *sync.Once
	memcacheClient *memcache.Client

	rabbitmqOnce       *sync.Once
	rabbitmqChannel    *amqp.Channel
	rabbitmqConnection *amqp.Connection
	rabbitError        error
}

func (d dependencies) getRedisClient(url string) (*redis.Client, error) {
	d.redisOnce.Do(func() {
		opt, err := redis.ParseURL(url)
		if err != nil {
			d.redisError = err
			return
		}
		d.redisClient = redis.NewClient(opt)
	})
	return d.redisClient, d.redisError
}

func (d dependencies) getMemcacheClient(url string) *memcache.Client {
	d.memcacheOnce.Do(func() {
		d.memcacheClient = memcache.New(url)
	})
	return d.memcacheClient
}

func (d dependencies) getNatsClient(url string) (*nats.Conn, error) {
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

func (d dependencies) getRabbitmqChannel(url string) (*amqp.Channel, error) {
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

func (d dependencies) Close() error {
	var err error
	if d.redisClient != nil {
		err = errors.Join(err, d.redisClient.Close())
	}
	if d.memcacheClient != nil {
		err = errors.Join(err, d.memcacheClient.Close())
	}
	if d.natsClient != nil {
		d.natsClient.Close()
	}
	if d.rabbitmqChannel != nil {
		err = errors.Join(err, d.rabbitmqChannel.Close(), d.rabbitmqConnection.Close())
	}
	return err
}

func CacheDriver(driver, url, prefix string, deps *dependencies) (cache.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		if err != nil {
			return nil, err
		}
		if err = redisClient.Ping(context.Background()).Err(); err != nil {
			return nil, err
		}
		return cache.NewRedisDriver(redisClient, prefix), nil
	case "memcached":
		memcacheClient := deps.getMemcacheClient(url)
		if err := memcacheClient.Ping(); err != nil {
			return nil, err
		}
		return cache.NewMemCachedDriver(memcacheClient, prefix), nil
	default:
		return cache.NewInMemoryDriver(), nil
	}
}

func EventDriver(driver, url string, queues []string, deps *dependencies) (bus.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		if err != nil {
			return nil, err
		}
		if err = redisClient.Ping(context.Background()).Err(); err != nil {
			return nil, err
		}
		return bus.NewRedisBroker(redisClient), nil
	case "nats":
		natsClient, err := deps.getNatsClient(url)
		return bus.NewNatsBroker(natsClient), err
	case "rabbitmq":
		return bus.NewRabbitBroker(url, queues)
	default:
		return bus.NewInMemoryDriver(queues), nil
	}
}

func JobSchedulerDriver(driver, url string, deps *dependencies) (scheduler.Driver, error) {
	switch driver {
	case "redis":
		redisClient, err := deps.getRedisClient(url)
		if err != nil {
			return nil, err
		}
		if err = redisClient.Ping(context.Background()).Err(); err != nil {
			return nil, err
		}
		return scheduler.NewRedisScheduler(redisClient, time.Second, 0), nil
	default:
		// return in memory driver instead of nil
		return nil, nil
	}
}
