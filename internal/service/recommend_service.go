package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"nodosml-pc4/internal/cache"
	"nodosml-pc4/internal/cluster"
	"nodosml-pc4/internal/models"
	"nodosml-pc4/internal/repository"
)

const (
	DefaultK = 20
	MaxK     = 50 // por seguridad, no deja pedir 1000 ítems
)

type RecommendService struct {
	ratings *repository.RatingRepository
	recRepo *repository.RecommendationRepository
	sims    *repository.SimilarityRepository
	// direcciones TCP de los nodos ML
	nodeAddrs []string
}

func NewRecommendService(
	r *repository.RatingRepository,
	recRepo *repository.RecommendationRepository,
	sims *repository.SimilarityRepository,
	nodeAddrs []string,
) *RecommendService {
	return &RecommendService{
		ratings:   r,
		recRepo:   recRepo,
		sims:      sims,
		nodeAddrs: nodeAddrs,
	}
}

// ====== Petición de recomendaciones (solo parámetros que sí cambian en runtime) ======

type RecRequest struct {
	UserID  int
	K       int
	Refresh bool
}

func cacheKey(req RecRequest) string {
	// Cachea por usuario + k (no incluye refresh, refresh solo decide si usar cache)
	return fmt.Sprintf("rec:user:%d:k:%d", req.UserID, req.K)
}

// Recommend: coordina el cluster de nodos ML
func (s *RecommendService) Recommend(ctx context.Context, req RecRequest) ([]models.RecItem, error) {
	// defaults y límites para K
	if req.K <= 0 {
		req.K = DefaultK
	} else if req.K > MaxK {
		req.K = MaxK
	}

	// 1) Cache Redis (solo si refresh = false)
	var cached []models.RecItem
	if !req.Refresh {
		if ok, err := cache.GetJSON(ctx, cacheKey(req), &cached); err == nil && ok {
			return cached, nil
		}
	}

	// 2) Ratings del usuario
	ratings, err := s.ratings.GetAllByUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if len(ratings) == 0 {
		return []models.RecItem{}, nil
	}

	if len(s.nodeAddrs) == 0 {
		return nil, fmt.Errorf("no ML nodes configured (ML_NODE_ADDRS vacío)")
	}
	shards := len(s.nodeAddrs)

	// 3) Preparar tareas para cada nodo
	tasks := make([]*cluster.RecTask, shards)
	for shardID := 0; shardID < shards; shardID++ {
		tasks[shardID] = &cluster.RecTask{
			UserID:  req.UserID,
			K:       req.K,
			ShardID: shardID,
			Shards:  shards,
			Ratings: ratings,
		}
	}

	// 4) Enviar en paralelo usando goroutines + channels
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resCh := make(chan *cluster.RecResponse, shards)
	errCh := make(chan error, shards)

	var wg sync.WaitGroup
	for i, addr := range s.nodeAddrs {
		wg.Add(1)
		go func(addr string, t *cluster.RecTask) {
			defer wg.Done()
			resp, err := cluster.SendTask(ctxTimeout, addr, t)
			if err != nil {
				errCh <- err
				return
			}
			resCh <- resp
		}(addr, tasks[i])
	}

	wg.Wait()
	close(resCh)
	close(errCh)

	if len(resCh) == 0 && len(errCh) > 0 {
		// si todos fallaron
		return nil, <-errCh
	}

	// 5) Combinar parciales: score = sum(num) / sum(den)
	scores := make(map[int]float64)
	weights := make(map[int]float64)

	for resp := range resCh {
		for _, p := range resp.Partials {
			scores[p.MovieID] += p.Num
			weights[p.MovieID] += p.Den
		}
	}

	var items []models.RecItem
	for mID, num := range scores {
		den := weights[mID]
		if den <= 0 {
			continue
		}
		items = append(items, models.RecItem{
			MovieID: mID,
			Score:   num / den,
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Score > items[j].Score })
	if len(items) > req.K {
		items = items[:req.K]
	}

	// 5.5) Guardar historial en Mongo (no rompemos la respuesta si falla)
	if s.recRepo != nil {
		hist := &models.Recommendation{
			UserID:           req.UserID,
			Algo:             "item-knn",
			SimilarityMetric: "cosine", // en esta PC4 usamos cosine fijo
			Params: map[string]any{
				"k":      req.K,
				"shards": shards,
				// aquí podrías agregar más cosas si luego cambias la lógica
				"refresh": req.Refresh,
			},
			Items:     items,
			CreatedAt: time.Now(),
		}

		if err := s.recRepo.Insert(ctx, hist); err != nil {
			log.Printf("error guardando recomendación en Mongo: %v", err)
		}
	}

	// 6) Cachear en Redis (1 hora)
	if err := cache.SetJSON(ctx, cacheKey(req), items, 60*60); err != nil {
		log.Printf("error cacheando recomendación en Redis: %v", err)
	}

	return items, nil
}

// ====== Explicación de una recomendación (item-based puro) ======

// Aquí sí tiene sentido usar algo/min_common/shrink porque se trabaja
// directamente sobre la colección de similitudes precalculadas.
type ExplainRequest struct {
	UserID    int
	MovieID   int
	Algo      string
	MinCommon int
	Shrink    int
}

// Explain reconstruye el score de una película recomendada
// en base a sus vecinos y los ratings del usuario.
func (s *RecommendService) Explain(ctx context.Context, req ExplainRequest) (*models.Explanation, error) {
	// defaults
	if req.Algo == "" {
		req.Algo = "cosine"
	}
	if req.MinCommon <= 0 {
		req.MinCommon = 5
	}
	if req.Shrink < 0 {
		req.Shrink = 0
	}

	// ratings del usuario
	ratings, err := s.ratings.GetAllByUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if len(ratings) == 0 {
		return nil, fmt.Errorf("el usuario %d no tiene ratings", req.UserID)
	}

	// mapa movieId -> rating del usuario
	ratingMap := make(map[int]float64, len(ratings))
	for _, r := range ratings {
		ratingMap[r.MovieID] = r.Rating
	}

	// vecinos de la película objetivo
	neighbors, err := s.sims.GetNeighbors(ctx, req.MovieID, 100)
	if err != nil {
		return nil, err
	}
	if len(neighbors) == 0 {
		return nil, fmt.Errorf("no hay vecinos precalculados para movieId=%d", req.MovieID)
	}

	var num, den float64
	var contribs []models.NeighborContribution

	for _, n := range neighbors {
		userRating, ok := ratingMap[n.MovieID]
		if !ok {
			continue // usuario no valoró a este vecino
		}
		if n.Sim <= 0 {
			continue
		}

		partial := n.Sim * userRating
		num += partial
		den += math.Abs(n.Sim)

		contribs = append(contribs, models.NeighborContribution{
			NeighborMovieID: n.MovieID,
			Sim:             n.Sim,
			UserRating:      userRating,
			Contribution:    partial, // luego normalizamos si quieres
		})
	}

	if den == 0 {
		return nil, fmt.Errorf("no se pudo explicar la recomendación (sin vecinos válidos)")
	}

	score := num / den

	// normalizar contribution a proporción
	for i := range contribs {
		if num != 0 {
			contribs[i].Contribution = contribs[i].Contribution / num
		}
	}

	exp := &models.Explanation{
		MovieID:   req.MovieID,
		Score:     score,
		Neighbors: contribs,
	}

	return exp, nil
}
