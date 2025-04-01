package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

type Route struct {
	Method  string
	Path    string
	Handler func(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error)
}

type Router struct {
	routes []Route
}

func (r *Router) AddRoute(method, path string, handler func(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error)) {
	r.routes = append(r.routes, Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
}

func (r *Router) FindRoute(method, path string) (*Route, bool) {

	for _, route := range r.routes {
		fmt.Printf("method %s : %s  path %s : %s \n", method, route.Method, path, route.Path)

		if route.Method == method && strings.HasPrefix(path, route.Path) {
			return &route, true
		}
	}
	return nil, false
}

func (s *Server) Route() func(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	return func(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
		router := &Router{}

		s.ApiKey(router)

		route, _ := router.FindRoute("GET", "/apikey/validate")
		return route.Handler(ctx, request)

	}
}
