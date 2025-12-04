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

// NextMovieID obtiene el siguiente movieId disponible (max + 1).
func (r *MovieRepository) NextMovieID(ctx context.Context) (int, error) {
	opts := options.FindOne().
		SetSort(bson.D{{Key: "movieId", Value: -1}}).
		SetProjection(bson.M{"movieId": 1})

	var m models.MovieDoc
	err := r.col.FindOne(ctx, bson.M{}, opts).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return 1, nil
	}
	if err != nil {
		return 0, err
	}
	return m.MovieID + 1, nil
}

// Insert inserta una nueva película.
func (r *MovieRepository) Insert(ctx context.Context, m *models.MovieDoc) error {
	_, err := r.col.InsertOne(ctx, m)
	return err
}

// Update reemplaza el documento completo de una película.
func (r *MovieRepository) Update(ctx context.Context, m *models.MovieDoc) error {
	_, err := r.col.ReplaceOne(ctx, bson.M{"movieId": m.MovieID}, m)
	return err
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

// ExistsByTitleYear indica si ya existe una película con ese título y año.
// Si year es nil, solo valida por título.
func (r *MovieRepository) ExistsByTitleYear(ctx context.Context, title string, year *int) (bool, error) {
	filter := bson.M{
		"title": title,
	}
	if year != nil {
		filter["year"] = *year
	}

	// usamos CountDocuments porque es simple y suficiente
	n, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
