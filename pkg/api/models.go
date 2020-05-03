package api

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Scan struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	URL       string             `json:"url" bson:"url"`
	JSON      string             `json:"json" bson:"json"`
	HTML      string             `json:"html" bson:"html"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
