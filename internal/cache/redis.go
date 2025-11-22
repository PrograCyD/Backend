package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"nodosml-pc4/internal/config"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

func InitRedis(cfg *config.Config) {
	client = redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("❌ Error conectando a Redis: %v", err)
	}

	log.Println("✅ Redis OK.")
}

// =======================================================
//  Helpers JSON para usar desde los servicios
// =======================================================

// GetJSON lee una key de Redis, si existe deserializa el JSON en `dest`.
func GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	if client == nil {
		return false, nil
	}

	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		// no existe la clave
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON serializa `value` a JSON y lo guarda en Redis con TTL en segundos.
func SetJSON(ctx context.Context, key string, value any, ttlSeconds int) error {
	if client == nil {
		return nil
	}

	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	ttl := time.Duration(ttlSeconds) * time.Second
	return client.Set(ctx, key, b, ttl).Err()
}
