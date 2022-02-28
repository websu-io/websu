package api

import (
	"context"
	"log"
	"os"
	"strings"
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

func TestValidateReportInvalidURL(t *testing.T) {
	r := ReportRequest{}
	r.URL = "https://notarealhost"
	err := r.Validate()
	if err == nil {
		t.Error("The URL is supposed to return an error")
	}
}

func TestValidateReportInvalidEmail(t *testing.T) {
	r := ReportRequest{}
	r.URL = "https://www.google.com"
	r.Email = "notavalidemail"
	err := r.Validate()
	if err == nil {
		t.Error("The invalid email is supposed to return an error")
	}
}

func TestValidateReport404Error(t *testing.T) {
	r := ReportRequest{}
	r.URL = "https://samos-it.com/thispagedoesnotexist"
	err := r.Validate()
	if !strings.Contains(err.Error(), "status code") {
		t.Error("Expected error that contained string status code")
	}
	if err == nil {
		t.Error("The URL is supposed to return an error")
	}
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
	if reports[0].User != "" {
		t.Errorf("Expected report.User to be set to '', but got %v", reports[0].User)
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

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "minute"
	r.LastRun = time.Now().Add(-58 * time.Second)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 7 {
		t.Errorf("Expected len(all) == 7, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 3 {
		t.Errorf("Expected len(reports) == 3, got len(reports) = %v", len(reports))
	}

	r = NewScheduledReport()
	r.URL = "https://www.google.com"
	r.Schedule = "minute"
	r.LastRun = time.Now().Add(-60 * time.Second)
	r.Insert()

	all, err = GetAllScheduledReports()
	if err != nil {
		t.Error(err.Error())
	}
	if len(all) != 8 {
		t.Errorf("Expected len(all) == 8, got len(all) = %v", len(all))
	}

	reports, err = GetScheduleReportsDueToRun()
	if err != nil {
		t.Error(err.Error())
	}
	if len(reports) != 4 {
		t.Errorf("Expected len(reports) == 3, got len(reports) = %v", len(reports))
	}

}

func TestLocations(t *testing.T) {
	l := NewLocation()
	l.Name = "local"
	l.DisplayName = "Local"
	l.Address = "lighthouse-server:50051"
	l.Order = 5
	l.Insert()

	lGet, err := GetLocationByName(l.Name)
	if err != nil {
		t.Error(err.Error())
	}
	if l.Name != lGet.Name {
		t.Errorf("Expected lGet.Name and l.Name to be equal. l.Name was %s and lGet.Name was %s", l.Name, lGet.Name)
	}

	rr := &ReportRequest{Location: l.Name}
	r := NewReportFromRequest(rr)
	if r.LocationDisplay != l.DisplayName {
		t.Errorf("Expected r.LocationDisplay and l.DisplayName to be equal. r.LocationDisplay was %s and l.DisplayName was %s",
			r.LocationDisplay, l.DisplayName)
	}
}

func TestLocationUpsertInvalidHex(t *testing.T) {
	_, err := NewLocationWithID("notvalidhex")
	if err == nil {
		t.Error("Error expected due to invalid hex")
	}
}

func TestLocationUpsert(t *testing.T) {
	l := NewLocation()
	l.Name = "local2"
	l.DisplayName = "Local"
	l.Address = "lighthouse-server:50051"
	l.Order = 5
	err := l.Upsert()
	if err != nil {
		t.Error(err.Error())
	}

	l.Order = 6
	l.Premium = true
	l.Upsert()

	lGet, err := GetLocationByName(l.Name)
	if err != nil {
		t.Error(err.Error())
	}
	if lGet.Order != l.Order {
		t.Error("Order wasn't updated")
	}
	if lGet.Premium != l.Premium {
		t.Error("Premium wasn't updated")
	}

}
