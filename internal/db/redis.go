package db

import (
	"context"

	"github.com/mnuddindev/devpulse/pkg/utils"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	*redis.Client
}

// NewRedis initializes a Redis client with context.
func NewRedis(ctx context.Context, addr, password string) (*RedisClient, error) {
	if err := ctx.Err(); err != nil {
		return nil, utils.WrapError(err, utils.ErrInternalServerError.Code, "redis initialization canceled")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})

	// Ping Redis with context
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, utils.NewError(utils.ErrInternalServerError.Code, "Failed to connect to Redis", err.Error())
	}

	return &RedisClient{client}, nil
}

// CloseRedis shuts down the Redis connection.
func (r *RedisClient) Close(logger *utils.Logger) error {
	if err := r.Client.Close(); err != nil {
		logger.Error(context.Background()).WithMeta(utils.Map{"error": err.Error()}).Log("Redis close failed")
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to close Redis", err.Error())
	}
	logger.Info(context.Background()).Log("Redis connection closed successfully")
	return nil
}
