package service

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"nodosml-pc4/internal/config"
	"nodosml-pc4/internal/db"
	"nodosml-pc4/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// documento auxiliar para el max(iIdx)
type maxIdxDoc struct {
	MaxIdx int `bson:"maxIdx"`
}

// AdminMaintenanceService orquesta el mantenimiento de mapeos/similitudes.
type AdminMaintenanceService struct {
	cfg     *config.Config
	mlNodes []string
}

// NewAdminMaintenanceService crea el servicio.
func NewAdminMaintenanceService(cfg *config.Config, mlNodes []string) *AdminMaintenanceService {
	return &AdminMaintenanceService{
		cfg:     cfg,
		mlNodes: mlNodes,
	}
}

// ---------------------- SUMMARY / PENDING ----------------------

// GetSimilaritySummary devuelve el resumen global.
func (s *AdminMaintenanceService) GetSimilaritySummary(
	ctx context.Context,
	minRatings int64,
) (*models.AdminSimilaritySummary, error) {

	mdb := db.DB()
	moviesColl := mdb.Collection("movies")
	simsColl := mdb.Collection("similarities")

	// Solo consideramos películas con al menos minRatings
	baseFilter := bson.M{
		"ratingStats.count": bson.M{"$gte": minRatings},
	}

	// total de películas con suficientes ratings
	totalMovies, err := moviesColl.CountDocuments(ctx, baseFilter)
	if err != nil {
		return nil, err
	}

	// películas con iIdx asignado
	withIdxFilter := bson.M{
		"iIdx":              bson.M{"$exists": true},
		"ratingStats.count": bson.M{"$gte": minRatings},
	}
	moviesWithIdx, err := moviesColl.CountDocuments(ctx, withIdxFilter)
	if err != nil {
		return nil, err
	}
	moviesWithoutIdx := totalMovies - moviesWithIdx
	if moviesWithoutIdx < 0 {
		moviesWithoutIdx = 0
	}

	// documentos de similitudes (coseno, k=20)
	simsFilter := bson.M{
		"metric": "cosine",
		"k":      20,
	}
	moviesWithSims, err := simsColl.CountDocuments(ctx, simsFilter)
	if err != nil {
		return nil, err
	}
	moviesWithoutSims := moviesWithIdx - moviesWithSims
	if moviesWithoutSims < 0 {
		moviesWithoutSims = 0
	}

	summary := &models.AdminSimilaritySummary{
		TotalMovies:               totalMovies,
		MoviesWithIdx:             moviesWithIdx,
		MoviesWithoutIdx:          moviesWithoutIdx,
		MoviesWithSimilarities:    moviesWithSims,
		MoviesWithoutSimilarities: moviesWithoutSims,
		MinRatings:                minRatings,
	}
	return summary, nil
}

// GetPendingSimilarities lista películas pendientes de mapeo / similitudes.
func (s *AdminMaintenanceService) GetPendingSimilarities(
	ctx context.Context,
	minRatings, limitWithoutIdx, limitWithoutSims int64,
) (*models.AdminPendingSimilarities, error) {

	mdb := db.DB()
	moviesColl := mdb.Collection("movies")

	// ---------- 1) Películas sin iIdx pero con suficientes ratings ----------
	var withoutIdx []models.PendingMovieWithoutIdx

	findFilter := bson.M{
		"iIdx":              bson.M{"$exists": false},
		"ratingStats.count": bson.M{"$gte": minRatings},
	}

	findOpts := options.Find().
		SetLimit(limitWithoutIdx).
		SetSort(bson.D{{Key: "ratingStats.count", Value: -1}}).
		SetProjection(bson.M{
			"movieId":           1,
			"title":             1,
			"ratingStats.count": 1,
		})

	cur, err := moviesColl.Find(ctx, findFilter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var doc struct {
			MovieID     int    `bson:"movieId"`
			Title       string `bson:"title"`
			RatingStats struct {
				Count int64 `bson:"count"`
			} `bson:"ratingStats"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		withoutIdx = append(withoutIdx, models.PendingMovieWithoutIdx{
			MovieID:      doc.MovieID,
			Title:        doc.Title,
			RatingsCount: doc.RatingStats.Count,
		})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	// ---------- 2) Películas con iIdx pero sin documento en similarities ----------
	var withoutSims []models.PendingMovieWithoutSims

	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "iIdx", Value: bson.D{{Key: "$exists", Value: true}}},
			{Key: "ratingStats.count", Value: bson.D{{Key: "$gte", Value: minRatings}}},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "similarities"},
			{Key: "localField", Value: "iIdx"},
			{Key: "foreignField", Value: "iIdx"},
			{Key: "as", Value: "sims"},
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "sims", Value: bson.D{{Key: "$size", Value: 0}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "movieId", Value: 1},
			{Key: "title", Value: 1},
			{Key: "iIdx", Value: 1},
			{Key: "ratingStats.count", Value: 1},
		}}},
		bson.D{{Key: "$sort", Value: bson.D{{Key: "ratingStats.count", Value: -1}}}},
		bson.D{{Key: "$limit", Value: limitWithoutSims}},
	}

	cur2, err := moviesColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur2.Close(ctx)

	for cur2.Next(ctx) {
		var doc struct {
			MovieID     int    `bson:"movieId"`
			Title       string `bson:"title"`
			IIdx        int    `bson:"iIdx"`
			RatingStats struct {
				Count int64 `bson:"count"`
			} `bson:"ratingStats"`
		}
		if err := cur2.Decode(&doc); err != nil {
			return nil, err
		}
		withoutSims = append(withoutSims, models.PendingMovieWithoutSims{
			MovieID:      doc.MovieID,
			IIdx:         doc.IIdx,
			Title:        doc.Title,
			RatingsCount: doc.RatingStats.Count,
		})
	}
	if err := cur2.Err(); err != nil {
		return nil, err
	}

	result := &models.AdminPendingSimilarities{
		MinRatings:          minRatings,
		WithoutIdx:          withoutIdx,
		WithoutSimilarities: withoutSims,
	}
	return result, nil
}

// ---------------------- REMAP MISSING ----------------------

// RemapMissingMovies asigna iIdx a las películas que aún no tienen.
func (s *AdminMaintenanceService) RemapMissingMovies(
	ctx context.Context,
	minRatings, limit int64,
) (*models.RemapMissingResult, error) {

	if limit <= 0 {
		limit = 1000
	}

	mdb := db.DB()
	moviesColl := mdb.Collection("movies")

	// 1) obtener max(iIdx) actual
	maxPipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "iIdx", Value: bson.D{{Key: "$type", Value: "int"}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "maxIdx", Value: bson.D{{Key: "$max", Value: "$iIdx"}}},
		}}},
	}

	cur, err := moviesColl.Aggregate(ctx, maxPipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	maxIdx := -1
	if cur.Next(ctx) {
		var doc maxIdxDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		maxIdx = doc.MaxIdx
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	// 2) buscar películas sin iIdx con >= minRatings ratings (máx 'limit')
	filter := bson.M{
		"iIdx":              bson.M{"$exists": false},
		"ratingStats.count": bson.M{"$gte": minRatings},
	}
	findOpts := options.Find().
		SetLimit(limit).
		SetSort(bson.D{{Key: "ratingStats.count", Value: -1}}).
		SetProjection(bson.M{"_id": 1})

	cur2, err := moviesColl.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur2.Close(ctx)

	var mappedCount int64
	fromIdx, toIdx := 0, 0
	nextIdx := maxIdx + 1

	for cur2.Next(ctx) {
		var doc struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		if err := cur2.Decode(&doc); err != nil {
			return nil, err
		}

		_, err := moviesColl.UpdateByID(ctx, doc.ID, bson.M{
			"$set": bson.M{"iIdx": nextIdx},
		})
		if err != nil {
			return nil, err
		}

		if mappedCount == 0 {
			fromIdx = nextIdx
		}
		toIdx = nextIdx
		nextIdx++
		mappedCount++
	}
	if err := cur2.Err(); err != nil {
		return nil, err
	}

	res := &models.RemapMissingResult{
		MappedCount: mappedCount,
		FromIdx:     fromIdx,
		ToIdx:       toIdx,
	}
	return res, nil
}

// ---------------------- REBUILD SIMILARITIES ----------------------

// RebuildSimilarities recalcula similitudes para películas sin doc en similarities.
func (s *AdminMaintenanceService) RebuildSimilarities(
	ctx context.Context,
	req *models.RebuildSimilaritiesRequest,
) (*models.RebuildSimilaritiesResult, error) {

	if req.BatchSize <= 0 {
		req.BatchSize = 50
	}
	if req.Parallelism <= 0 {
		req.Parallelism = 4
	}
	if len(s.mlNodes) == 0 {
		return nil, errors.New("no hay nodos ML configurados")
	}

	mdb := db.DB()
	moviesColl := mdb.Collection("movies")

	// 1) Buscar películas con iIdx pero sin similarities y con minRatings.
	var pendingIIdxs []int

	pipeline := bson.A{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "iIdx", Value: bson.D{{Key: "$exists", Value: true}}},
			{Key: "ratingStats.count", Value: bson.D{{Key: "$gte", Value: req.MinRatings}}},
		}}},
		bson.D{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "similarities"},
			{Key: "localField", Value: "iIdx"},
			{Key: "foreignField", Value: "iIdx"},
			{Key: "as", Value: "sims"},
		}}},
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "sims", Value: bson.D{{Key: "$size", Value: 0}}},
		}}},
		bson.D{{Key: "$project", Value: bson.D{
			{Key: "iIdx", Value: 1},
		}}},
	}

	cur, err := moviesColl.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var doc struct {
			IIdx int `bson:"iIdx"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		pendingIIdxs = append(pendingIIdxs, doc.IIdx)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	if len(pendingIIdxs) == 0 {
		return &models.RebuildSimilaritiesResult{
			ProcessedMovies: 0,
			Batches:         0,
			K:               req.K,
			MinCommonUsers:  req.MinCommonUsers,
			Shrink:          req.Shrink,
		}, nil
	}

	// 2) Particionar en batches.
	var batches [][]int
	for i := 0; i < len(pendingIIdxs); i += req.BatchSize {
		j := i + req.BatchSize
		if j > len(pendingIIdxs) {
			j = len(pendingIIdxs)
		}
		batches = append(batches, pendingIIdxs[i:j])
	}

	// 3) Ejecutar batches en paralelo contra los nodos ML.
	var wg sync.WaitGroup
	sem := make(chan struct{}, req.Parallelism)
	errCh := make(chan error, len(batches))

	for idx, batch := range batches {
		sem <- struct{}{}
		wg.Add(1)

		go func(batchNum int, b []int) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := s.callMLNodeForBatch(ctx, batchNum, b, req); err != nil {
				errCh <- err
				return
			}
		}(idx, batch)
	}

	wg.Wait()
	close(errCh)

	if len(errCh) > 0 {
		// por simplicidad devolvemos el primer error
		return nil, <-errCh
	}

	result := &models.RebuildSimilaritiesResult{
		ProcessedMovies: len(pendingIIdxs),
		Batches:         len(batches),
		K:               req.K,
		MinCommonUsers:  req.MinCommonUsers,
		Shrink:          req.Shrink,
	}
	return result, nil
}

// callMLNodeForBatch manda un batch de iIdxs a cualquier nodo ML (round-robin simple).
func (s *AdminMaintenanceService) callMLNodeForBatch(
	ctx context.Context,
	batchNum int,
	iIdxs []int,
	req *models.RebuildSimilaritiesRequest,
) error {

	// De momento lo dejamos como stub: solo selecciona el nodo, arma la URL
	// y no hace el POST real (para no romper mientras montas el endpoint).
	_ = ctx
	node := s.mlNodes[batchNum%len(s.mlNodes)]
	_ = node

	// Aquí luego implementarás el POST real al nodo ML.

	_ = (&http.Client{Timeout: 120 * time.Second})
	return nil
}
