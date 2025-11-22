package models

import "time"

type RecItem struct {
	MovieID int     `bson:"movieId" json:"movieId"`
	Score   float64 `bson:"score"  json:"score"`
}

type Recommendation struct {
	ID               string    `bson:"_id,omitempty"        json:"id"`
	UserID           int       `bson:"userId"               json:"userId"`
	Algo             string    `bson:"algo"                 json:"algo"`
	SimilarityMetric string    `bson:"similarityMetric"     json:"similarityMetric"`
	Params           any       `bson:"params"               json:"params"`
	Items            []RecItem `bson:"items"                json:"items"`
	CreatedAt        time.Time `bson:"createdAt"            json:"createdAt"`
}

// ====== Explicación de una recomendación (para /recommendations/explain) ======

type NeighborContribution struct {
	NeighborMovieID int     `json:"neighbor_movie_id" bson:"neighbor_movie_id"`
	Sim             float64 `json:"sim"               bson:"sim"`
	UserRating      float64 `json:"user_rating"       bson:"user_rating"`
	Contribution    float64 `json:"contribution"      bson:"contribution"`
}

type Explanation struct {
	MovieID   int                    `json:"movie_id" bson:"movie_id"`
	Score     float64                `json:"score"    bson:"score"`
	Neighbors []NeighborContribution `json:"neighbors" bson:"neighbors"`
}
