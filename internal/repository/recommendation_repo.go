package repository

import (
	"context"
	"time"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/mongo"
)

type RecommendationRepository struct {
	col *mongo.Collection
}

func NewRecommendationRepository() *RecommendationRepository {
	return &RecommendationRepository{
		col: db.DB().Collection("recommendations"),
	}
}

func (r *RecommendationRepository) Insert(ctx context.Context, rec *models.Recommendation) error {
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	_, err := r.col.InsertOne(ctx, rec)
	return err
}

// opcional para futuro: listar historial por usuario
func (r *RecommendationRepository) FindByUser(ctx context.Context, userID int, limit int64) ([]models.Recommendation, error) {
	cur, err := r.col.Find(ctx, map[string]any{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.Recommendation
	for cur.Next(ctx) {
		var rec models.Recommendation
		if err := cur.Decode(&rec); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, cur.Err()
}
