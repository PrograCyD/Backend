// internal/repository/movie_repo.go
package repository

import (
	"context"

	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MovieRepository struct {
	col *mongo.Collection
}

func NewMovieRepository() *MovieRepository {
	return &MovieRepository{col: db.DB().Collection("movies")}
}

func (r *MovieRepository) GetByID(ctx context.Context, movieID int) (*models.MovieDoc, error) {
	var m models.MovieDoc
	err := r.col.FindOne(ctx, bson.M{"movieId": movieID}).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &m, err
}

func (r *MovieRepository) Search(
	ctx context.Context,
	q string,
	genre string,
	yearFrom, yearTo int,
	limit, offset int,
) ([]models.MovieDoc, error) {

	filter := bson.M{}

	if q != "" {
		filter["title"] = bson.M{"$regex": q, "$options": "i"}
	}
	if genre != "" {
		// géneros es un array, esto busca que contenga ese género
		filter["genres"] = genre
	}
	if yearFrom > 0 || yearTo > 0 {
		yearCond := bson.M{}
		if yearFrom > 0 {
			yearCond["$gte"] = yearFrom
		}
		if yearTo > 0 {
			yearCond["$lte"] = yearTo
		}
		filter["year"] = yearCond
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(offset))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.MovieDoc
	for cur.Next(ctx) {
		var m models.MovieDoc
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, cur.Err()
}

// Top por popularidad (count) o rating promedio
func (r *MovieRepository) Top(ctx context.Context, metric string, limit int) ([]models.MovieDoc, error) {
	sortField := "ratingStats.count" // popular
	if metric == "rating" {
		sortField = "ratingStats.average"
	}

	opts := options.Find().
		SetSort(bson.D{{Key: sortField, Value: -1}}).
		SetLimit(int64(limit))

	cur, err := r.col.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.MovieDoc
	for cur.Next(ctx) {
		var m models.MovieDoc
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, cur.Err()
}
