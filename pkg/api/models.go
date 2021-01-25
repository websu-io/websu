package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"reflect"
	"time"
)

var (
	DB           *mongo.Client
	DatabaseName = "websu"
)

func CreateMongoClient(mongoURI string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var err error
	DB, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	} else {
		log.Info("Connected to Database")
	}
	CreateMongoIndexes()
}

type ReportRequest struct {
	// Required parameter the URL of the website
	URL string `json:"url" bson:"url" example:"https://www.google.com"`
	// Optional parameter, possible values are desktop or mobile. If unset will default to desktop
	FormFactor string `json:"form_factor" bson:"form_factor" example:"desktop"`
	// Optional parameter, by default will be set to 1000 if omitted
	ThroughputKbps int64 `json:"throughput_kbps" bson:"thoughput_kbps" example:"50000"`
	// Optional parameter, default location will be used if not set
	Location string `json:"location" bson:"location" example:"australia-southeast1"`
}

func checkLocation(value interface{}) error {
	s, _ := value.(string)
	if s == "" {
		return nil
	}
	if _, ok := LighthouseClients[s]; ok == false {
		return fmt.Errorf("Specified location doesn't exist. Possible values are: %v",
			reflect.ValueOf(LighthouseClients).MapKeys())
	}
	return nil
}

func (r ReportRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.URL, validation.Required, is.URL),
		validation.Field(&r.FormFactor, validation.In("desktop", "mobile")),
		validation.Field(&r.ThroughputKbps, validation.Min(1000), validation.Max(500000)),
		validation.Field(&r.Location, validation.By(checkLocation)),
	)
}

type ScheduledReport struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	ReportRequest `bson:",inline"`
	Schedule      string    `json:"schedule" bson:"schedule" example:"daily"`
	LastRun       time.Time `json:"last_run" bson:"last_run"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
}

func (s ScheduledReport) Validate() error {
	return validation.ValidateStruct(&s,
		validation.Field(&s.ReportRequest),
		validation.Field(&s.Schedule, validation.In("hourly", "daily", "weekly", "monthly")),
	)
}

type Report struct {
	ID            primitive.ObjectID `json:"id" bson:"_id"`
	ReportRequest `bson:",inline"`
	// RawJSON contains the lighthouse JSON result
	RawJSON          string                 `json:"raw_json" bson:"raw_json"`
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

type Location struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name" example:"australia-southeast1"`
	DisplayName string             `json:"display_name" bson:"display_name" example:"Sydney, AU"`
	Address     string             `json:"address" bson:"address" example:"8.8.8.8:50051"`
	// Flag to indicate whether TLS should be used
	Secure    bool      `json:"secure" bson:"secure"`
	Order     int32     `json:"order" bson:"order"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

func CreateMongoIndexes() {
	locationsIndexOpts := mongo.IndexModel{
		Keys:    bson.M{"name": 1},
		Options: options.Index().SetUnique(true),
	}
	locations := DB.Database(DatabaseName).Collection("locations")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	locIndexName, err := locations.Indexes().CreateOne(ctx, locationsIndexOpts)
	if err != nil {
		log.WithError(err).Error("Error creating mongoDB locations indexe")
	}
	log.WithField("name", locIndexName).Info("Created index for locations")

	reportsIndex := mongo.IndexModel{
		Keys:    bson.M{"created_at": -1},
		Options: nil,
	}
	reports := DB.Database(DatabaseName).Collection("reports")
	reportsIndexName, err := reports.Indexes().CreateOne(ctx, reportsIndex)
	if err != nil {
		log.WithError(err).Error("Error creating mongoDB reports index")
	}
	log.WithField("name", reportsIndexName).Info("Created index for reports")
}

func GetAllReports(limit int64, skip int64) ([]Report, error) {
	reports := []Report{}
	collection := DB.Database(DatabaseName).Collection("reports")
	c := context.TODO()
	options := options.Find()
	options.SetProjection(bson.M{"raw_json": 0, "audit_results": 0})
	options.SetSort(bson.M{"created_at": -1})
	options.SetLimit(limit)
	options.SetSkip(skip)
	cursor, err := collection.Find(c, bson.D{}, options)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &reports); err != nil {
		return nil, err
	}
	return reports, nil
}

func GetAllReportsCount() (int64, error) {
	collection := DB.Database(DatabaseName).Collection("reports")
	opts := options.EstimatedDocumentCount().SetMaxTime(2 * time.Second)
	count, err := collection.EstimatedDocumentCount(context.Background(), opts)
	if err != nil {
		return 0, err
	}
	return count, nil
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
	collection := DB.Database(DatabaseName).Collection("reports")
	if _, err := collection.InsertOne(ctx, report); err != nil {
		return err
	}
	return nil
}

func (report *Report) Delete() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := DB.Database(DatabaseName).Collection("reports").DeleteOne(context.TODO(), bson.M{"_id": report.ID}, nil)
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
	collection := DB.Database(DatabaseName).Collection("reports")
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

func NewLocation() *Location {
	l := new(Location)
	l.ID = primitive.NewObjectID()
	l.CreatedAt = time.Now()
	return l
}

func (location *Location) Insert() error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	collection := DB.Database(DatabaseName).Collection("locations")
	if _, err := collection.InsertOne(ctx, location); err != nil {
		return err
	}
	return nil
}

func (location *Location) Delete() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := DB.Database(DatabaseName).Collection("locations").DeleteOne(context.TODO(), bson.M{"_id": location.ID}, nil)
	if err != nil {
		return err
	}
	if result.DeletedCount == 1 {
		return nil
	} else if result.DeletedCount == 0 {
		return errors.New("Location with id " + location.ID.Hex() + " did not exist")
	} else {
		return errors.New("Multiple locations were deleted.")
	}
	return nil
}

func GetAllLocations() ([]Location, error) {
	locations := []Location{}
	collection := DB.Database(DatabaseName).Collection("locations")
	c := context.TODO()
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{"order", 1}})
	cursor, err := collection.Find(c, bson.D{}, findOptions)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &locations); err != nil {
		return nil, err
	}
	return locations, nil
}

func GetLocationByObjectIDHex(hex string) (Location, error) {
	var location Location
	collection := DB.Database(DatabaseName).Collection("locations")
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return location, err
	}
	err = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&location)
	if err != nil {
		return location, err
	}
	return location, nil
}

func NewScheduledReport() *ScheduledReport {
	s := new(ScheduledReport)
	s.ID = primitive.NewObjectID()
	s.CreatedAt = time.Now()
	return s
}

func (sr *ScheduledReport) Insert() error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	collection := DB.Database(DatabaseName).Collection("scheduled_reports")
	if _, err := collection.InsertOne(ctx, sr); err != nil {
		return err
	}
	return nil
}

func (sr *ScheduledReport) Update() error {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	filter := bson.M{"_id": sr.ID}
	collection := DB.Database(DatabaseName).Collection("scheduled_reports")
	if _, err := collection.ReplaceOne(ctx, filter, sr); err != nil {
		return err
	}
	return nil
}

func (sr *ScheduledReport) Delete() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := DB.Database(DatabaseName).Collection("scheduled_reports").DeleteOne(context.TODO(), bson.M{"_id": sr.ID}, nil)
	if err != nil {
		return err
	}
	if result.DeletedCount == 1 {
		return nil
	} else if result.DeletedCount == 0 {
		return errors.New("ScheduledReport with id " + sr.ID.Hex() + " did not exist")
	} else {
		return errors.New("Multiple ScheduledReports were deleted.")
	}
	return nil
}

func GetAllScheduledReports() ([]ScheduledReport, error) {
	scheduledReports := []ScheduledReport{}
	collection := DB.Database(DatabaseName).Collection("scheduled_reports")
	c := context.TODO()
	cursor, err := collection.Find(c, bson.D{})
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &scheduledReports); err != nil {
		return nil, err
	}
	return scheduledReports, nil
}

func GetScheduledReportByObjectIDHex(hex string) (ScheduledReport, error) {
	var sr ScheduledReport
	collection := DB.Database(DatabaseName).Collection("scheduled_reports")
	oid, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		return sr, err
	}
	// db.sales.aggregate( [ { $project: { item: 1, dateDifference: { $subtract: [ "$$NOW", "$date" ] } } } ] )
	err = collection.FindOne(context.Background(), bson.M{"_id": oid}).Decode(&sr)
	if err != nil {
		return sr, err
	}
	return sr, nil
}

func GetScheduleReportsDueToRun() ([]ScheduledReport, error) {
	scheduledReports := []ScheduledReport{}
	collection := DB.Database(DatabaseName).Collection("scheduled_reports")
	c := context.TODO()
	query := bson.M{
		"$or": []bson.M{
			bson.M{"schedule": "hourly",
				"last_run": bson.M{
					"$lte": time.Now().Add(-60 * time.Minute)}},
			bson.M{"schedule": "daily",
				"last_run": bson.M{
					"$lte": time.Now().AddDate(0, 0, -1)}},
			bson.M{"schedule": "weekly",
				"last_run": bson.M{
					"$lte": time.Now().AddDate(0, 0, -7)}},
			bson.M{"schedule": "monthly",
				"last_run": bson.M{
					"$lte": time.Now().AddDate(0, -1, 0)}},
		}}
	cursor, err := collection.Find(c, query)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(c, &scheduledReports); err != nil {
		return nil, err
	}
	return scheduledReports, nil
}
