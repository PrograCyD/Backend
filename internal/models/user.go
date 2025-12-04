package models

type UserDoc struct {
	UserID int  `json:"userId" bson:"userId"`
	UIdx   *int `json:"uIdx,omitempty" bson:"uIdx,omitempty"`

	FirstName string `json:"firstName,omitempty" bson:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty" bson:"lastName,omitempty"`
	Username  string `json:"username,omitempty" bson:"username,omitempty"`

	Email        string `json:"email" bson:"email"`
	PasswordHash string `json:"passwordHash" bson:"passwordHash"`
	Role         string `json:"role" bson:"role"`

	About           string   `json:"about,omitempty" bson:"about,omitempty"`
	PreferredGenres []string `json:"preferredGenres,omitempty" bson:"preferredGenres,omitempty"`

	CreatedAt string `json:"createdAt" bson:"createdAt"`
	UpdatedAt string `json:"updatedAt" bson:"updatedAt"`
}
