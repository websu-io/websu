package api

import (
	"context"
	"errors"
	"github.com/jinzhu/copier"
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

type ReportRequest struct {
	URL            string `json:"url" example:"https://www.google.com"`
	FormFactor     string `json:"form_factor" example:"desktop"`
	ThroughputKbps int64  `json:"throughput_kbps" example:"50000"`
}

type Report struct {
	ID               primitive.ObjectID     `json:"id" bson:"_id"`
	URL              string                 `json:"url" bson:"url"`
	FormFactor       string                 `json:"form_factor" bson:"form_factor" example:"desktop"`
	ThroughputKbps   int64                  `json:"throughput_kbps" example:"50000"`
	RawJSON          string                 `json:"raw_json" bson:"-"`
	CreatedAt        time.Time              `json:"created_at" bson:"created_at"`
	PerformanceScore float32                `json:"performance_score" bson:"performance_score"`
	AuditResults     map[string]AuditResult `json:"audit_results" bson:"audit_results"`
}

type AuditResult struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Score            float32 `json:"score"`
	ScoreDisplayMode string  `json:"scoreDisplayMode"`
	NumericValue     float64 `json:"numericValue"`
	NumericUnit      string  `json:"numericUnit"`
	DisplayValue     string  `json:"DisplayValue"`
}

func GetAllReports() ([]Report, error) {
	reports := []Report{}
	collection := DB.Database("websu").Collection("reports")
	c := context.TODO()
	cursor, err := collection.Find(c, bson.D{})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &reports); err != nil {
		return nil, err
	}
	return reports, nil

}

func NewReport() *Report {
	r := new(Report)
	r.ID = primitive.NewObjectID()
	r.CreatedAt = time.Now()
	return r
}

func NewReportFromRequest(rr *ReportRequest) *Report {
	r := NewReport()
	copier.Copy(&r, rr)
	return r
}

func (report *Report) Insert() error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	collection := DB.Database("websu").Collection("reports")
	if _, err := collection.InsertOne(ctx, report); err != nil {
		return err
	}
	return nil
}

func (report *Report) Delete() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := DB.Database("websu").Collection("reports").DeleteOne(context.TODO(), bson.M{"_id": report.ID}, nil)
	if err != nil {
		return err
	}
	if result.DeletedCount == 1 {
		return nil
	} else if result.DeletedCount == 0 {
		return errors.New("Report with id " + report.ID.Hex() + " did not exist")
	} else {
		return errors.New("Multiple reports were deleted.")
	}
	return nil
}

func GetReportByObjectIDHex(hex string) (Report, error) {
	var report Report
	collection := DB.Database("websu").Collection("reports")
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return report, err
	}
	err = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&report)
	if err != nil {
		return report, err
	}
	return report, nil
}
