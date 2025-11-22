package models

type Neighbor struct {
	MovieID int     `json:"movieId" bson:"movieId"`
	IIdx    int     `json:"iIdx" bson:"iIdx"`
	Sim     float64 `json:"sim" bson:"sim"`
}

type SimilarityDoc struct {
	ID        string     `json:"_id" bson:"_id"`
	MovieID   int        `json:"movieId" bson:"movieId"`
	IIdx      int        `json:"iIdx" bson:"iIdx"`
	Metric    string     `json:"metric" bson:"metric"`
	K         int        `json:"k" bson:"k"`
	Neighbors []Neighbor `json:"neighbors" bson:"neighbors"`
	UpdatedAt string     `json:"updatedAt" bson:"updatedAt"`
}
