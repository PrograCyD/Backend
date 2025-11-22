package cluster

import "nodosml-pc4/internal/models"

// Tarea enviada desde el coordinador (API) a cada nodo ML.
type RecTask struct {
	UserID  int                `json:"userId"`
	K       int                `json:"k"`
	ShardID int                `json:"shardId"` // id del shard (0..Shards-1)
	Shards  int                `json:"shards"`  // total de shards/nodos
	Ratings []models.RatingDoc `json:"ratings"`
}

// Parcial de score: no devolvemos score final, sino numerador y denominador
// para que el coordinador combine correctamente entre shards.
type PartialScore struct {
	MovieID int     `json:"movieId"`
	Num     float64 `json:"num"` // sum(sim * rating)
	Den     float64 `json:"den"` // sum(sim)
}

// Respuesta de un nodo ML a la API.
type RecResponse struct {
	ShardID  int            `json:"shardId"`
	Partials []PartialScore `json:"partials"`
}
