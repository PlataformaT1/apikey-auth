package redis

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient     *redis.Client
	redisClientOnce sync.Once
)

// GetClient returns a singleton Redis client
func GetClient() *redis.Client {
	redisClientOnce.Do(func() {
		// Get Redis connection details from environment variables
		redisHost := os.Getenv("USER_VAR_REDIS_HOST")
		if redisHost == "" {
			redisHost = "localhost:6379" // Default for local development
		}

		//redisPassword := os.Getenv("REDIS_PASSWORD")

		// Create Redis client
		redisClient = redis.NewClient(&redis.Options{
			Addr:     redisHost,
			Password: "",
			DB:       0, // Default DB
		})

		// Test the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := redisClient.Ping(ctx).Result()
		if err != nil {
			log.Printf("Failed to connect to Redis: %v", err)
		} else {
			log.Printf("Successfully connected to Redis at %s", redisHost)
		}
	})

	return redisClient
}

// CheckAndIncrementRateLimit checks if the seller has exceeded their rate limit
// and increments the counter if not
func CheckAndIncrementRateLimit(ctx context.Context, sellerID string, maxRequestsPerSecond int) (bool, error) {
	rdb := GetClient()

	// Create a timestamp for the current second
	now := time.Now().Unix()
	key := fmt.Sprintf("ratelimit:%s:%d", sellerID, now)

	// Use a Lua script to ensure atomicity
	script := `
	local current = redis.call('INCR', KEYS[1])
	if current == 1 then
		redis.call('EXPIRE', KEYS[1], 1)
	end
	return current
	`

	// Run the script
	result, err := rdb.Eval(ctx, script, []string{key}).Int64()
	if err != nil {
		log.Printf("Redis error: %v", err)
		// In case of Redis error, we'll allow the request to proceed
		return true, err
	}

	// Check if rate limit is exceeded
	return int(result) <= maxRequestsPerSecond, nil
}
