package main

import (
	"apikey/internal/api/handler"
	"apikey/internal/api/server"
	mongodb "apikey/internal/repository"
	mysql "apikey/internal/repository"
	"apikey/internal/service"
	"apikey/pkg/env"
	"apikey/pkg/logger"
	"log"
)

func init() {
	if err := env.Validate(env.GetEnvs()); err != nil {
		log.Fatal(err)
	}

	logger, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := mongodb.Connection()
	if err != nil {
		log.Fatal(err)
	}

	companyRepository := mysql.NewApiKeyRepository(logger, db, "api_key_db", "api_key")
	companyService := service.NewApiKeyService(logger, companyRepository)
	companyHandler := handler.NewApiKeyHandler(logger, companyService)

	svr = server.New(
		server.WithApiKeyHandler(companyHandler),
	)
}
