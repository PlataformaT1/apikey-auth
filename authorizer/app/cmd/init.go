package main

import (
	"apikey/pkg/redis"
)

func init() {

	// This will initialize the Redis connection
	_ = redis.GetClient()
}
