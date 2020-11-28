package api

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

var DB *mongo.Client

func CreateMongoClient(mongoURI string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	DB, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Connected to Database")
	}
}

type Scan struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	URL       string             `json:"url" bson:"url"`
	Json      string             `json:"json" bson:"-"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}

func GetAllScans() ([]Scan, error) {
	scans := []Scan{}
	collection := DB.Database("websu").Collection("scans")
	c := context.TODO()
	cursor, err := collection.Find(c, bson.D{})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &scans); err != nil {
		return nil, err
	}
	return scans, nil

}

func NewScan() *Scan {
	s := new(Scan)
	s.ID = primitive.NewObjectID()
	s.CreatedAt = time.Now()
	return s
}

func (scan *Scan) Insert() error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	collection := DB.Database("websu").Collection("scans")
	if _, err := collection.InsertOne(ctx, scan); err != nil {
		return err
	}
	return nil
}

func (scan *Scan) Delete() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := DB.Database("websu").Collection("scans").DeleteOne(context.TODO(), bson.M{"_id": scan.ID}, nil)
	if err != nil {
		return err
	}
	if result.DeletedCount == 1 {
		return nil
	} else if result.DeletedCount == 0 {
		return errors.New("Scan with id " + scan.ID.Hex() + " did not exist")
	} else {
		return errors.New("Multiple scans were deleted.")
	}
	return nil
}

func GetScanByObjectIDHex(hex string) (Scan, error) {
	var scan Scan
	collection := DB.Database("websu").Collection("scans")
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return scan, err
	}
	err = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&scan)
	if err != nil {
		return scan, err
	}
	return scan, nil

}
