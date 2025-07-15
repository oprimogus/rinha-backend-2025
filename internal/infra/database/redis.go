package database

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	"github.com/oprimogus/rinha-backend-2025/internal/config"
)

var rdb *Redis

type Redis struct {
	*redis.Client
}

func NewRedis(host string, port int, password string) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       0,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &Redis{client}, nil
}

func GetRedis() *Redis {
    if rdb == nil {
        cfg := config.GetInstance()
        instance, err := NewRedis(cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.Password)
        if err != nil {
            panic(err)
        }
        rdb = instance
    }
    return rdb
}
