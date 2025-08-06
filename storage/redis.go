package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// RedisStore, Redis istemcisini ve context'i tutar.
type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

const (
	leaderboardKey = "leaderboard"
)

// NewRedisStore, Redis'e bağlanır ve yeni bir RedisStore örneği döndürür
func NewRedisStore(ctx context.Context, redisAddr string) (*RedisStore, error) {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("redis'e bağlanılamadı: %w", err)
	}
	log.Printf("Redis sunucusuna başarıyla bağlanıldı: %s", redisAddr)
	return &RedisStore{client: rdb, ctx: ctx}, nil
}

// AddScore, bir oyuncunun skorunu Redis'teki sıralamaya ekler
func (rs *RedisStore) AddScore(playerID string, score int) error {
	return rs.client.ZAdd(rs.ctx, leaderboardKey, redis.Z{
		Score:  float64(score),
		Member: playerID,
	}).Err()
}

// GetLeaderboard, Redis'ten sıralamadaki ilk 10 oyuncuyu skorlarıyla birlikte döndürür
func (rs *RedisStore) GetLeaderboard() ([]redis.Z, error) {
	return rs.client.ZRevRangeWithScores(rs.ctx, leaderboardKey, 0, 9).Result()
}
