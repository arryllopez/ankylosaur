package ankylogo

import (
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	redisConnect *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{redisConnect: client}
}
