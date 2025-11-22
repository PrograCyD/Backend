package models

// Lo que está en Mongo (igual a tu NDJSON)
type RatingDoc struct {
	UserID    int     `json:"userId" bson:"userId"`
	MovieID   int     `json:"movieId" bson:"movieId"`
	Rating    float64 `json:"rating" bson:"rating"`
	Timestamp int64   `json:"timestamp" bson:"timestamp"`
}

// Lo que devolvemos por API (mismo formato, pero separado por claridad)
type Rating struct {
	UserID    int     `json:"userId"`
	MovieID   int     `json:"movieId"`
	Rating    float64 `json:"rating"`
	Timestamp int64   `json:"timestamp"`
}
