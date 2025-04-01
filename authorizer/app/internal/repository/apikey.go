package mongodb

import (
	"apikey/internal/errormap"
	"apikey/internal/model/apikey"
	"context"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type ApiKeyRepository interface {
	ValidateApiKey(ctx context.Context, apiKey string) (*apikey.ApiKey, error)
}

type apikeyRepository struct {
	logger     *logrus.Logger
	collection *mongo.Collection
}

func NewApiKeyRepository(logger *logrus.Logger, db *mongo.Client, dbName, collectionName string) ApiKeyRepository {
	collection := db.Database(dbName).Collection(collectionName)
	return &apikeyRepository{
		logger:     logger,
		collection: collection,
	}
}

func (u *apikeyRepository) ValidateApiKey(ctx context.Context, apiKey string) (*apikey.ApiKey, error) {
	var dbApiKey apikey.ApiKey
	filter := bson.M{"apiKey": apiKey}

	err := u.collection.FindOne(ctx, filter).Decode(&dbApiKey)
	if err == mongo.ErrNoDocuments {
		u.logger.Infof("ApiKey  %s not found", apiKey)
		return nil, errormap.ErrNoRows
	} else if err != nil {
		u.logger.Errorf("error retrieving ApiKey : %v", err)
		return nil, err
	}

	return &dbApiKey, nil
}
