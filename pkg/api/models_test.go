package api

import (
	"context"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	cmd := exec.Command("docker-compose", "up", "-d", "mongo")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("Starting mongo docker container had an error. Out: %s, Err: %s", string(out), err)
	}

	CreateMongoClient("mongodb://localhost:27017")
	DatabaseName = "websu-test"
	code := m.Run()
	ctx := context.TODO()
	if err := DB.Database(DatabaseName).Drop(ctx); err != nil {
		log.Fatalf("Error dropping test database %v: %v", DatabaseName, err)
	}
	os.Exit(code)
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
