package storage

import (
	"context"

	"github.com/mnuddindev/devpulse/pkg/logger"
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
func (r *RedisClient) Close(log *logger.Logger) error {
	if err := r.Client.Close(); err != nil {
		log.Error(context.Background()).WithMeta(map[string]string{"error": err.Error()}).Logs("Redis close failed")
		return utils.NewError(utils.ErrInternalServerError.Code, "Failed to close Redis", err.Error())
	}
	log.Info(context.Background()).Logs("Redis connection closed successfully")
	return nil
}
