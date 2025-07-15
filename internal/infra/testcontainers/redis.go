package testcontainers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func MakeRedis(ctx context.Context) (*Container, error) {
    redisContainer, err := redis.Run(ctx, "redis:8.0.3-alpine3.21")
    
    if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to start container: %s", err))
		return nil, err
	}
	
	hostPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		slog.ErrorContext(ctx, fmt.Sprintf("failed to get mapped port: %s", err))
		return nil, err
	}
	port := strings.ReplaceAll(string(hostPort), "/tcp", "")
	
    return &Container{
        name: "redis",
        instance: redisContainer,
        Port: port,
    }, nil
}