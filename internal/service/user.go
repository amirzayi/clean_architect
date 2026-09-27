package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/amirzayi/clean_architect/internal/domain"
	"github.com/amirzayi/clean_architect/internal/repository"
	"github.com/amirzayi/clean_architect/pkg/bus"
	"github.com/amirzayi/clean_architect/pkg/cache"
	"github.com/amirzayi/clean_architect/pkg/errs"
	"github.com/amirzayi/clean_architect/pkg/paginate"
	"github.com/amirzayi/clean_architect/pkg/synq"
)

type User interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context, pagination *paginate.Pagination) ([]domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, user domain.User) error
}

type user struct {
	db       repository.User
	cache    *cache.Cache
	eventBus *bus.EventBus
	logger   *slog.Logger
	dbcache  synq.CacheSync
}

func NewUserService(db repository.User, cache *cache.Cache, eventBus *bus.EventBus, logger *slog.Logger) User {
	return &user{
		db:       db,
		cache:    cache,
		eventBus: eventBus,
		logger:   logger,
		dbcache:  synq.New(cache, logger),
	}
}

func (u *user) Create(ctx context.Context, user domain.User) (domain.User, error) {
	user.ID = uuid.New()
	user.Status = domain.UsereStatusNew
	user.CreatedAt = time.Now()

	err := u.dbcache.SetAsync(user.ID.String(), user, func() error {
		return u.db.Create(ctx, user)
	}, time.Hour)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			return domain.User{}, errs.New(err, errs.CodeExisted)
		}
		u.logger.Error("failed to create user", slog.Any("error", err))
		return domain.User{}, errs.New(err, errs.CodeInternal)
	}
	return user, nil
}

func (u *user) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := u.dbcache.GetAsync(ctx, email, func() (domain.User, error) {
		return u.db.GetByEmail(ctx, email)
	}, time.Minute)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return domain.User{}, errs.NotFound("user")
		}
		u.logger.Error("failed to get user by email", slog.Any("error", err))
		return domain.User{}, errs.New(err, errs.CodeInternal)
	}
	return user, nil
}

func (u *user) List(ctx context.Context, pagination *paginate.Pagination) ([]domain.User, error) {
	users, err := u.db.List(ctx, pagination)
	if err != nil {
		u.logger.Error("failed to list users", slog.Any("error", err))
		return nil, errs.New(err, errs.CodeInternal)
	}
	return users, nil
}

func (u *user) Delete(ctx context.Context, id uuid.UUID) error {
	err := u.dbcache.DeleteAsync(id.String(), func() error {
		return u.db.Delete(ctx, id)
	})
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return errs.NotFound("user")
		}
		u.logger.Error("failed to delete user", slog.Any("error", err))
		return errs.New(err, errs.CodeInternal)
	}
	return nil
}

func (u *user) Update(ctx context.Context, user domain.User) error {
	err := u.dbcache.SetAsync(user.ID.String(), user, func() error {
		return u.db.Update(ctx, user)
	}, time.Minute)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return errs.NotFound("user")
		}
		u.logger.Error("failed to update user", slog.Any("error", err))
		return errs.New(err, errs.CodeInternal)
	}
	return nil
}

func (u *user) GetByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := u.dbcache.GetAsync(ctx, id.String(), func() (domain.User, error) {
		return u.db.GetByID(ctx, id)
	}, time.Minute)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return user, errs.NotFound("user")
		}
		u.logger.Error("failed to get user by id", slog.Any("error", err))
		return domain.User{}, errs.New(err, errs.CodeInternal)
	}
	return user, nil
}
