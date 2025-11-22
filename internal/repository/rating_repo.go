package repository

import (
	"context"
	"time"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RatingRepository struct {
	col *mongo.Collection
}

func NewRatingRepository() *RatingRepository {
	return &RatingRepository{col: db.DB().Collection("ratings")}
}

func (r *RatingRepository) UpsertRating(ctx context.Context, userID, movieID int, rating float64) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"userId": userID, "movieId": movieID},
		bson.M{"$set": bson.M{
			"rating": rating,
			// guardamos epoch (int64)
			"timestamp": time.Now().Unix(),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

// helpers de casteo seguro
func asInt(v any) int {
	switch x := v.(type) {
	case int32:
		return int(x)
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int32:
		return int64(x)
	case int64:
		return x
	case float64:
		return int64(x)
	default:
		return 0
	}
}

func asFloat64(v any) float64 {
	switch x := v.(type) {
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float64:
		return x
	default:
		return 0
	}
}

func (r *RatingRepository) GetByUser(ctx context.Context, userID, limit, offset int) ([]models.RatingDoc, error) {
	cur, err := r.col.Find(ctx,
		bson.M{"userId": userID},
		options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)),
	)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.RatingDoc
	for cur.Next(ctx) {
		var raw bson.M
		if err := cur.Decode(&raw); err != nil {
			return nil, err
		}

		rd := models.RatingDoc{
			UserID:    asInt(raw["userId"]),
			MovieID:   asInt(raw["movieId"]),
			Rating:    asFloat64(raw["rating"]),
			Timestamp: asInt64(raw["timestamp"]),
		}
		out = append(out, rd)
	}
	return out, cur.Err()
}

func (r *RatingRepository) GetAllByUser(ctx context.Context, userID int) ([]models.RatingDoc, error) {
	return r.GetByUser(ctx, userID, 10000, 0)
}
