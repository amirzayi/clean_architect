package mongodb

import (
	"context"

	"github.com/AmirMirzayi/clean_architecture/internal/user/adapter/repository"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/AmirMirzayi/clean_architecture/internal/user/domain"
)

var _ repository.Repository = mongoRepository{}

type mongoRepository struct {
	collection *mongo.Collection
}

func NewRepository(collection *mongo.Collection) repository.Repository {
	return mongoRepository{
		collection: collection,
	}
}

func (r mongoRepository) Create(ctx context.Context, user domain.User) error {
	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return err
	}
	return nil
}
