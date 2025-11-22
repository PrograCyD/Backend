// internal/service/movie_service.go
package service

import (
	"context"

	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"
)

type MovieService struct {
	movies *repository.MovieRepository
}

func NewMovieService(m *repository.MovieRepository) *MovieService {
	return &MovieService{movies: m}
}

func (s *MovieService) GetMovie(ctx context.Context, id int) (*models.MovieDoc, error) {
	return s.movies.GetByID(ctx, id)
}

func (s *MovieService) Search(
	ctx context.Context,
	q, genre string,
	yearFrom, yearTo, limit, offset int,
) ([]models.MovieDoc, error) {
	return s.movies.Search(ctx, q, genre, yearFrom, yearTo, limit, offset)
}

func (s *MovieService) Top(ctx context.Context, metric string, limit int) ([]models.MovieDoc, error) {
	return s.movies.Top(ctx, metric, limit)
}
