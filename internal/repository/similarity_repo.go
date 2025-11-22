package repository

import (
	"context"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type SimilarityRepository struct {
	col *mongo.Collection
}

func NewSimilarityRepository() *SimilarityRepository {
	return &SimilarityRepository{col: db.DB().Collection("similarities")}
}

// Devuelve los vecinos (Neighbor) de una película por movieId, truncando a k.
func (r *SimilarityRepository) GetNeighbors(ctx context.Context, movieID, k int) ([]models.Neighbor, error) {
	var doc models.SimilarityDoc
	err := r.col.FindOne(ctx, bson.M{"movieId": movieID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return []models.Neighbor{}, nil
	}
	if err != nil {
		return nil, err
	}

	neighbors := doc.Neighbors
	if len(neighbors) > k {
		neighbors = neighbors[:k]
	}
	return neighbors, nil
}
