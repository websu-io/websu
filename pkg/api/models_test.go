package api

import (
	"context"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	CreateMongoClient("mongodb://localhost:27018")
	DatabaseName = "websu-test"
	code := m.Run()
	ctx := context.TODO()
	if err := DB.Database(DatabaseName).Drop(ctx); err != nil {
		log.Fatalf("Error dropping test database %v: %v", DatabaseName, err)
	}
	os.Exit(code)
}

func TestGetReports(t *testing.T) {
	r := NewReport()
	r.URL = "https://www.sam.com"
	r.User = "sam"
	r.Insert()

	r = NewReport()
	r.URL = "https://www.google.com"
	r.Insert()

	r = NewReport()
	r.URL = "https://www.google.com"
	r.User = "rob"
	r.Insert()
	query := map[string]interface{}{"user": "sam"}
	reports, err := GetReports(10, 0, query)
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 1 {
		t.Errorf("len(reports) should be 1, but got %v", len(reports))
	}
	if reports[0].User != "sam" {
		t.Errorf("Expected report.User to be set to sam, but got %v", reports[0].User)
	}

	reports, err = GetReports(10, 0, nil)
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 3 {
		t.Errorf("len(reports) should be 3, but got %v", len(reports))
	}
}

func TestScheduledReports(t *testing.T) {
	r := NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "daily"
	r.Insert()
	all, err := GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 1 {
		t.Errorf("Expected len(all) == 1, got len(all) = %v", len(all))
	}
	reports, err := GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 1 {
		t.Errorf("Expected len(reports) == 1, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "daily"
	r.LastRun = time.Now().Add(time.Hour * 23)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 2 {
		t.Errorf("Expected len(all) == 2, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 1 {
		t.Errorf("Expected len(reports) == 1, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "weekly"
	r.LastRun = time.Now().AddDate(0, 0, -6)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 3 {
		t.Errorf("Expected len(all) == 3, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 1 {
		t.Errorf("Expected len(reports) == 1, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "weekly"
	r.LastRun = time.Now().AddDate(0, 0, -8)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 4 {
		t.Errorf("Expected len(all) == 4, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 2 {
		t.Errorf("Expected len(reports) == 2, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "hourly"
	r.LastRun = time.Now().Add(-59 * time.Minute)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 5 {
		t.Errorf("Expected len(all) == 5, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 2 {
		t.Errorf("Expected len(reports) == 2, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "hourly"
	r.LastRun = time.Now().Add(-60 * time.Minute)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 6 {
		t.Errorf("Expected len(all) == 6, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 3 {
		t.Errorf("Expected len(reports) == 3, got len(reports) = %v", len(reports))
	}
}
