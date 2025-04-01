package main

/*
import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-lambda-go/events"
)

func TestGetCompanyHandler(t *testing.T) {
	// Configura el manejador y la función de limpieza

	svr, err := run()
	if err != nil {
		log.Fatalf("error runnig lambda server: %v", err)
	}
	// Define una solicitud de API Gateway simulada
	input := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/test",
	}

	// Invoca el manejador Lambda
	_, err = svr.Route()(context.Background(), input)
	if err != nil {
		t.Fatalf("error invoking get commerce: %v", err)
	}

}

func TestEnvMising(t *testing.T) {
	// Valida que la función devolvió un error al validar las envs al tener setupEnv
	os.Setenv("USER_VAR_DB_HOST", "")
	_, err := run()
	if err == nil {
		t.Error("expected env error")
	}

}

func TestLoggerMisingConfig(t *testing.T) {
	// Valida que la función devolvió un error el logger
	os.Setenv("USER_VAR_LOG_LEVEL", "INCORRECTLOGLEVEL")

	_, err := run()

	if err == nil {
		t.Error("expected env error")
	}

}

func TestDbConnectionFail(t *testing.T) {
	os.Setenv("USER_VAR_DB_HOST", "invalid-host")
	// Valida que la función devolvió un error al validar las envs
	_, err := run()
	if err == nil {
		t.Error("expected db fail error")
	}

}
*/
