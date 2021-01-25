package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/websu-io/websu/pkg/api"
	"github.com/websu-io/websu/pkg/lighthouse"
	"github.com/websu-io/websu/pkg/mocks"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

var a *api.App

func TestMain(m *testing.M) {
	a = api.NewApp()
	api.CreateMongoClient("mongodb://localhost:27018")
	api.DatabaseName = "websu-test"
	code := m.Run()
	ctx := context.TODO()
	if err := api.DB.Database(api.DatabaseName).Drop(ctx); err != nil {
		log.Fatalf("Error dropping test database %v: %v", api.DatabaseName, err)
	}
	os.Exit(code)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	a.Router.ServeHTTP(rr, req)
	return rr
}

func checkResponseCode(t *testing.T, expected int, response *httptest.ResponseRecorder) {
	if expected != response.Code {
		t.Errorf("Expected response code %d. Got %d\n. Body: %s", expected, response.Code, response.Body.String())
	}
}

func deleteAllReports() {

	reports, err := api.GetReports(500, 0, nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, report := range reports {
		report.Delete()
	}
	a = api.NewApp()
	api.CreateMongoClient("mongodb://localhost:27018")
	api.DatabaseName = "websu-test"
}

func createReport(t *testing.T, body []byte, mockLighthouseServer bool) *httptest.ResponseRecorder {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLightHouseClient := mocks.NewMockLighthouseServiceClient(ctrl)
	api.LighthouseClient = mockLightHouseClient
	report := bytes.NewBuffer(body)
	req, _ := http.NewRequest("POST", "/reports", report)
	if mockLighthouseServer {
		mockLightHouseClient.EXPECT().Run(gomock.Any(), gomock.Any()).Return(
			&lighthouse.LighthouseResult{Stdout: []byte("{}")}, nil,
		)
	}
	resp := executeRequest(req)
	return resp
}

func TestGetReportsEmpty(t *testing.T) {
	deleteAllReports()
	req, _ := http.NewRequest("GET", "/reports", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("Expected an empty array as []. Got %s", body)
	}
}

func TestGetReportsLimitSkip(t *testing.T) {
	deleteAllReports()
	for i := 1; i <= 50; i++ {
		r := api.NewReport()
		r.URL = fmt.Sprintf("http://test-%v", i)
		r.CreatedAt = time.Now().AddDate(0, 0, i)
		if err := r.Insert(); err != nil {
			t.Error("Error inserting report: " + err.Error())
		}
	}
	req, _ := http.NewRequest("GET", "/reports", nil)
	resp := executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	var reports []api.Report
	if err := json.NewDecoder(resp.Body).Decode(&reports); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, resp.Body)
	}
	if len(reports) != 50 {
		t.Errorf("Expected len(reports) = 50, but got len(reports) = %v", len(reports))
	}

	req, _ = http.NewRequest("GET", "/reports?limit=10", nil)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	var partialReports []api.Report
	if err := json.NewDecoder(resp.Body).Decode(&partialReports); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, resp.Body)
	}
	if len(partialReports) != 10 {
		t.Errorf("Expected len(reports) = 10, but got len(reports) = %v", len(partialReports))
	}
	for i, report := range partialReports {
		if report.URL != reports[i].URL {
			t.Errorf("Expected reports to be the same. Expected %v, but got %v", report.URL, reports[i].URL)
		}
	}

	req, _ = http.NewRequest("GET", "/reports?limit=10&skip=10", nil)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	partialReports = nil
	if err := json.NewDecoder(resp.Body).Decode(&partialReports); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, resp.Body)
	}
	if len(partialReports) != 10 {
		t.Errorf("Expected len(reports) = 10, but got len(reports) = %v", len(partialReports))
	}
	for i, report := range partialReports {
		if report.URL != reports[i+10].URL {
			t.Errorf("Expected reports to be the same. Expected %v, but got %v", report.URL, reports[i+10].URL)
		}
	}
}

func TestCreateReport(t *testing.T) {
	body := []byte(`{"URL": "https://www.google.com"}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "google.com") != true {
		t.Errorf("Expected body to contain google.com. Got %s", body)
	}
	deleteAllReports()
}

func TestCreateReportRateLimit(t *testing.T) {
	body := []byte(`{"URL": "https://www.google.com"}`)
	var mock bool
	var responseCode int
	for i := 1; i < 12; i++ {
		if i >= 11 {
			mock = false
			responseCode = http.StatusTooManyRequests
		} else {
			mock = true
			responseCode = http.StatusOK
		}
		response := createReport(t, body, mock)
		checkResponseCode(t, responseCode, response)
	}
	deleteAllReports()
}

func TestCreateReportFFDesktop(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "form_factor": "desktop"}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "google.com") != true {
		t.Errorf("Expected body to contain google.com. Got %s", body)
	}
	if body := response.Body.String(); strings.Contains(body, "\"form_factor\":\"desktop\"") != true {
		t.Errorf("Expected body to form_factor: 'desktop'. Got %s", body)
	}
	deleteAllReports()
}

func TestCreateReportFFMobile(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "form_factor": "mobile"}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "google.com") != true {
		t.Errorf("Expected body to contain google.com. Got %s", body)
	}
	if body := response.Body.String(); strings.Contains(body, "\"form_factor\":\"mobile\"") != true {
		t.Errorf("Expected body to form_factor: 'mobile'. Got %s", body)
	}
	deleteAllReports()
}

func TestCreateReportFFMInvalid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "form_factor": "invalid"}`)
	response := createReport(t, body, false)
	checkResponseCode(t, http.StatusBadRequest, response)
	expected := "form_factor: must be a valid value"
	if body := response.Body.String(); strings.Contains(body, expected) != true {
		t.Errorf("Expected body to contain %s. Got %s", expected, body)
	}
	deleteAllReports()
}

func TestCreateReportThroughputKbpsValid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "throughput_kbps": 50000}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	deleteAllReports()
}

func TestCreateReportThroughputKbpsInvalid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "throughput_kbps": "not a number"}`)
	response := createReport(t, body, false)
	checkResponseCode(t, http.StatusBadRequest, response)
	deleteAllReports()
}

func TestCreateGetandDeleteReport(t *testing.T) {
	body := []byte(`{"URL": "https://www.google.com"}`)
	r := createReport(t, body, true)

	checkResponseCode(t, http.StatusOK, r)

	var report api.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, r.Body)
	}
	if report.URL != "https://www.google.com" {
		t.Errorf("Error: report.URL should be: https://www.google.com, but got %v", report.URL)
	}
	req, _ := http.NewRequest("GET", "/reports/"+report.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)
	report = api.Report{}
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, r.Body)
	}
	if report.URL != "https://www.google.com" {
		t.Errorf("Error: report.URL should be: https://www.google.com, but got %v", report.URL)
	}
	req, _ = http.NewRequest("GET", "/reports", nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)
	var reports []api.Report
	if err := json.NewDecoder(r.Body).Decode(&reports); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, r.Body)
	}
	if reports[0].URL != "https://www.google.com" {
		t.Errorf("Error: report.URL should be: https://www.google.com, but got %v", report.URL)
	}

	req, _ = http.NewRequest("DELETE", "/reports/"+report.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)

	deleteAllReports()
}

func TestDeleteReportNonExisting(t *testing.T) {
	// not a valid hex string
	req, _ := http.NewRequest("DELETE", "/reports/doesnotexist", nil)
	log.Printf("Request: %+v", req)
	r := executeRequest(req)
	checkResponseCode(t, http.StatusBadRequest, r)
	log.Printf("Response: %+v", r)

	// valid hex string but doesnt exist
	req, _ = http.NewRequest("DELETE", "/reports/5eab5a25b830c33d857dc045", nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusBadRequest, r)
	log.Printf("Response: %+v", r)

}

func TestGetLocationsEmpty(t *testing.T) {
	req, _ := http.NewRequest("GET", "/locations", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("Expected an empty array as []. Got %s", body)
	}
}

func TestCreateLocationsAndOrder(t *testing.T) {
	body := []byte(`{
		"address": "localhost:50051",
		"secure": false,
		"name": "local",
		"display_name": "Local",
		"order": 1
	}`)
	location := bytes.NewBuffer(body)
	req, _ := http.NewRequest("POST", "/locations", location)
	resp := executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if body := resp.Body.String(); strings.Contains(body, "localhost:50051") != true {
		t.Errorf("Expected body to contain localhost:50051. Got %s", body)
	}

	body = []byte(`{
		"address": "test2:443",
		"secure": true,
		"name": "test2",
		"display_name": "Remote",
		"order": 3
	}`)
	location = bytes.NewBuffer(body)
	req, _ = http.NewRequest("POST", "/locations", location)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if body := resp.Body.String(); strings.Contains(body, "test2:443") != true {
		t.Errorf("Expected body to contain test2:443. Got %s", body)
	}

	body = []byte(`{
		"address": "order2:443",
		"secure": true,
		"name": "test3",
		"display_name": "Remote",
		"order": 2
	}`)
	location = bytes.NewBuffer(body)
	req, _ = http.NewRequest("POST", "/locations", location)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if body := resp.Body.String(); strings.Contains(body, "order2:443") != true {
		t.Errorf("Expected body to contain order2:443. Got %s", body)
	}

	locations, err := api.GetAllLocations()
	if err != nil {
		t.Error(err)
	}
	if len(locations) != 3 {
		t.Errorf("Expected len(locations) = 1. Got len(locations) = %v", len(locations))
	}
	if locations[1].Address != "order2:443" {
		t.Error("Expected order2 to be the 2nd element. Ordering seems to be wrong.")
	}
}

func TestCreateLocationsDuplicateName(t *testing.T) {
	body := []byte(`{
		"address": "localhost:50051",
		"secure": false,
		"name": "unique123",
		"display_name": "Local",
		"order": 1
	}`)
	location := bytes.NewBuffer(body)
	req, _ := http.NewRequest("POST", "/locations", location)
	resp := executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if body := resp.Body.String(); strings.Contains(body, "localhost:50051") != true {
		t.Errorf("Expected body to contain localhost:50051. Got %s", body)
	}
	body = []byte(`{
		"address": "localhost:50051",
		"secure": false,
		"name": "unique123",
		"display_name": "Local",
		"order": 1
	}`)
	location = bytes.NewBuffer(body)
	req, _ = http.NewRequest("POST", "/locations", location)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusInternalServerError, resp)
}

func TestGetScheduledReportsEmpty(t *testing.T) {
	req, _ := http.NewRequest("GET", "/scheduled-reports", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("Expected an empty array as []. Got %s", body)
	}
}

func TestCreateScheduledReportAndDelete(t *testing.T) {
	body := []byte(`{
		"url": "https://www.google.com",
		"schedule": "daily"
	}`)
	srJson := bytes.NewBuffer(body)
	req, _ := http.NewRequest("POST", "/scheduled-reports", srJson)
	resp := executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if body := resp.Body.String(); strings.Contains(body, "https://www.google.com") != true {
		t.Errorf("Expected body to contain https://www.google.com. Got %s", body)
	}

	var sr api.ScheduledReport
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, resp.Body)
	}
	var srs []api.ScheduledReport
	req, _ = http.NewRequest("GET", "/scheduled-reports", nil)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
	if err := json.NewDecoder(resp.Body).Decode(&srs); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, resp.Body)
	}
	if srs[0].URL != "https://www.google.com" {
		t.Errorf("Expected url https://www.google.com but got %v", srs[0].URL)
	}
	if srs[0].Schedule != "daily" {
		t.Errorf("Expected schedule to be daily  but got %v", srs[0].Schedule)
	}

	req, _ = http.NewRequest("GET", "/scheduled-reports/"+sr.ID.Hex(), srJson)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)

	req, _ = http.NewRequest("DELETE", "/scheduled-reports/"+sr.ID.Hex(), srJson)
	resp = executeRequest(req)
	checkResponseCode(t, http.StatusOK, resp)
}

func TestCreateScheduledReportInvalidSchedule(t *testing.T) {
	body := []byte(`{
		"url": "https://www.google.com",
		"schedule": "invalidschedule"
	}`)
	sr := bytes.NewBuffer(body)
	req, _ := http.NewRequest("POST", "/scheduled-reports", sr)
	resp := executeRequest(req)
	checkResponseCode(t, http.StatusBadRequest, resp)
	if body := resp.Body.String(); strings.Contains(body, "schedule: must be a valid value") != true {
		t.Errorf("Expected schedule: must be a valid value. Got %s", body)
	}
}

func TestHTTPRunReport(t *testing.T) {
	ts := httptest.NewServer(a.Router)
	defer ts.Close()
	api.ApiUrl = ts.URL
	rr := api.ReportRequest{URL: "https://www.google.com"}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLightHouseClient := mocks.NewMockLighthouseServiceClient(ctrl)
	api.LighthouseClient = mockLightHouseClient
	mockLightHouseClient.EXPECT().Run(gomock.Any(), gomock.Any()).Return(
		&lighthouse.LighthouseResult{Stdout: []byte("{}")}, nil,
	)
	api.HTTPRunReport(rr)
}
