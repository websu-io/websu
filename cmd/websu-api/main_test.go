package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/websu-io/websu/pkg/api"
	"github.com/websu-io/websu/pkg/lighthouse"
	"github.com/websu-io/websu/pkg/mocks"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

var a *api.App

func TestMain(m *testing.M) {
	cmd := exec.Command("docker-compose", "up", "-d", "mongo")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Starting mongo docker container had an error: %v", err)
	}
	cmd = exec.Command("docker", "run", "-p", "127.0.0.1:6379:6379",
		"--name", "websu-redis", "-d", "redis")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Starting redis docker container had an error: %v", err)
	}

	a = api.NewApp()
	api.CreateMongoClient("mongodb://localhost:27017")
	api.DatabaseName = "websu-test"
	code := m.Run()
	ctx := context.TODO()
	if err := api.DB.Database(api.DatabaseName).Drop(ctx); err != nil {
		log.Fatalf("Error dropping test database %v: %v", api.DatabaseName, err)
	}

	cmd = exec.Command("docker-compose", "stop", "mongo")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Stopping mongo docker container had an error: %v", err)
	}

	cmd = exec.Command("docker", "stop", "websu-redis")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Stopping websu-redis docker container had an error: %v", err)
	}
	cmd = exec.Command("docker", "rm", "websu-redis")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Deleting websu-redis docker container had an error: %v", err)
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

func cleanUpAfterTest() {

	reports, err := api.GetAllReports()
	if err != nil {
		log.Fatal(err)
	}
	for _, report := range reports {
		report.Delete()
	}
	a = api.NewApp()
	api.CreateMongoClient("mongodb://localhost:27017")
	api.DatabaseName = "websu-test"
}

func TestGetReportsEmpty(t *testing.T) {
	cleanUpAfterTest()
	req, _ := http.NewRequest("GET", "/reports", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("Expected an empty array as []. Got %s", body)
	}
}

func createReport(t *testing.T, body []byte, mockLighthouseServer bool) *httptest.ResponseRecorder {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLightHouseClient := mocks.NewMockLighthouseServiceClient(ctrl)
	a.LighthouseClient = mockLightHouseClient
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

func TestCreateReport(t *testing.T) {
	body := []byte(`{"URL": "https://www.google.com"}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "google.com") != true {
		t.Errorf("Expected body to contain google.com. Got %s", body)
	}
	cleanUpAfterTest()
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
	cleanUpAfterTest()
}

func TestCreateReportRateLimitRedis(t *testing.T) {
	a = api.NewApp(api.WithRedis("redis://localhost:6379/0"))
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
	cleanUpAfterTest()
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
	cleanUpAfterTest()
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
	cleanUpAfterTest()
}

func TestCreateReportFFMInvalid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "form_factor": "invalid"}`)
	response := createReport(t, body, false)
	checkResponseCode(t, http.StatusBadRequest, response)
	if body := response.Body.String(); strings.Contains(body, "Invalid form_factor") != true {
		t.Errorf("Expected body to contain Invalid form_factor. Got %s", body)
	}
	cleanUpAfterTest()
}

func TestCreateReportThroughputKbpsValid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "throughput_kbps": 50000}`)
	response := createReport(t, body, true)
	checkResponseCode(t, http.StatusOK, response)
	cleanUpAfterTest()
}

func TestCreateReportThroughputKbpsInvalid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "throughput_kbps": "not a number"}`)
	response := createReport(t, body, false)
	checkResponseCode(t, http.StatusBadRequest, response)
	cleanUpAfterTest()
}

func TestCreateGetandDeleteReport(t *testing.T) {
	body := []byte(`{"URL": "https://www.google.com"}`)
	r := createReport(t, body, true)

	checkResponseCode(t, http.StatusOK, r)

	var report api.Report
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, r.Body)
	}

	req, _ := http.NewRequest("GET", "/reports/"+report.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)

	req, _ = http.NewRequest("DELETE", "/reports/"+report.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)

	cleanUpAfterTest()
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
