package main

import (
	"apikey/internal/api/server"

	_ "github.com/go-sql-driver/mysql"

	"github.com/aws/aws-lambda-go/lambda"
)

var svr *server.Server

func main() {

	lambda.Start(svr.Route())

}
