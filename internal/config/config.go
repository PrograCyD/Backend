package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	MongoDB    string
	RedisAddr  string
	RedisPass  string
	JWTSecret  string
	HTTPPort   string
	TMDBAPIKey string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		MongoURI:   getEnv("MONGO_URI", "mongodb://root:example@localhost:27017"),
		MongoDB:    getEnv("MONGO_DB", "pc4_movies"),
		RedisAddr:  getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPass:  getEnv("REDIS_PASSWORD", ""),
		JWTSecret:  getEnv("JWT_SECRET", "super-secret"),
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		TMDBAPIKey: getEnv("TMDB_API_KEY", "5f947eefe9278165015da465d0af58c3"),
	}
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Printf("[config] %s no está seteado, usando valor por defecto\n", key)
		return def
	}
	return v
}
