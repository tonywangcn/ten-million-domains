package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
)

var (
	Redis *redis.Client
	ctx   = context.TODO()
)

func init() {
	env := os.Getenv("ENV")
	redisHost := os.Getenv("REDIS_HOST")
	redisPort, err := strconv.Atoi(os.Getenv("REDIS_PORT"))
	if err != nil {
		redisPort = 6379 // default Redis port if not set
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")
	log.Printf("Initializing Redis: %s, env: %s, port: %d", redisHost, env, redisPort)
	if env == "PROD" {
		option := &redis.FailoverOptions{
			MasterName:       "mymaster",
			SentinelAddrs:    []string{fmt.Sprintf("%s:%d", redisHost, redisPort)},
			Password:         redisPassword,
			SentinelPassword: redisPassword,
			DB:               0,
		}
		Redis = redis.NewFailoverClient(option)
	} else {
		option := &redis.Options{
			Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
			Password: redisPassword,
			DB:       0,
		}
		if redisPassword != "" {
			option.Password = redisPassword
		}
		Redis = redis.NewClient(option)
	}

	pong, err := Redis.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Errorf("failed to inititialize Redis %s", err.Error()))
	}
	log.Printf("Redis Initialized %s", pong)
}

func FlushAll() error {
	return Redis.FlushAll(ctx).Err()
}

func Incr(key string) error {
	return Redis.Incr(ctx, key).Err()
}

func IncrBy(key string, count int64) error {
	return Redis.IncrBy(ctx, key, count).Err()
}

func Decr(key string) error {
	return Redis.Decr(ctx, key).Err()
}

func SAdd(key string, values []string) error {
	return Redis.SAdd(ctx, key, values).Err()
}

func SPop(key string) string {
	return Redis.SPop(ctx, key).Val()
}

func Scard(key string) int64 {
	return Redis.SCard(ctx, key).Val()
}

// SPopN pops a specified number of items from a Redis set.
func SPopN(key string, n int) []string {
	var results []string

	// Using a pipeline for efficiency
	pipe := Redis.Pipeline()

	// Queue up SPOP commands for `n` times
	for i := 0; i < n; i++ {
		pipe.SPop(ctx, key)
	}

	// Execute the pipeline and get the results
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return results
	}

	// Process each command result to collect the popped items
	for _, cmder := range cmders {
		if spopCmd, ok := cmder.(*redis.StringCmd); ok {
			val, err := spopCmd.Result()
			if err == nil && val != "" {
				results = append(results, val)
			}
		}
	}

	return results
}
