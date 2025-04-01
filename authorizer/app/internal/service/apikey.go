package service

import (
	"apikey/internal/errormap"
	"apikey/internal/model/apikey"
	"apikey/pkg/errorx"

	mysql "apikey/internal/repository"
	"context"

	"github.com/sirupsen/logrus"
)

type ServiceApiKey interface {
	ValidateApiKey(ctx context.Context, apikey string) (*apikey.ApiKey, error)
}

type serviceApiKey struct {
	logger           *logrus.Logger
	apikeyRepository mysql.ApiKeyRepository
}

func NewApiKeyService(logger *logrus.Logger, apikeyRepository mysql.ApiKeyRepository) ServiceApiKey {
	return &serviceApiKey{
		logger:           logger,
		apikeyRepository: apikeyRepository,
	}
}

var (
	ErrInvalidArgument = errorx.NewErrorf(errormap.CodeInvalidArgument, "invalid params")
	ErrInvalidEmail    = errorx.NewErrorf(errormap.CodeInvalidArgument, "invalid commerce")
)

func (s *serviceApiKey) ValidateApiKey(ctx context.Context, id string) (*apikey.ApiKey, error) {
	resp, err := s.apikeyRepository.ValidateApiKey(ctx, id)
	if err != nil {
		s.logger.Errorf("Error getting apikey: %v", err)
		return &apikey.ApiKey{}, err
	}

	err = resp.Validate()

	if err != nil {
		s.logger.Errorf("Invalid apikey: %v", err)
		return &apikey.ApiKey{}, err
	}

	return resp, nil
}
