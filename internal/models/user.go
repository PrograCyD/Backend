package models

type UserDoc struct {
	UserID       int    `json:"userId" bson:"userId"`
	UIdx         *int   `json:"uIdx,omitempty" bson:"uIdx,omitempty"`
	Email        string `json:"email" bson:"email"`
	PasswordHash string `json:"passwordHash" bson:"passwordHash"`
	Role         string `json:"role" bson:"role"`
	CreatedAt    string `json:"createdAt" bson:"createdAt"`
}
