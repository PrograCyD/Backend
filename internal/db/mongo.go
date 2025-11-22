package db

import (
	"context"
	"log"
	"time"

	"nodosml-pc4/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClient *mongo.Client
var mongoDB *mongo.Database

func InitMongo(cfg *config.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("[mongo] error conectando: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("[mongo] ping falló: %v", err)
	}

	mongoClient = client
	mongoDB = client.Database(cfg.MongoDB)
	log.Printf("[mongo] conectado a %s / DB=%s\n", cfg.MongoURI, cfg.MongoDB)
}

func DB() *mongo.Database {
	return mongoDB
}
