package main

import (
	"bytes"
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
	a = api.NewApp()

	cmd := exec.Command("docker-compose", "up", "-d", "mongo")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Starting mongo docker container had an error: %v", err)
	}

	api.CreateMongoClient("mongodb://localhost:27017")
	code := m.Run()

	cmd = exec.Command("docker-compose", "stop", "mongo")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Stopping mongo docker container had an error: %v", err)
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

func dbClearReports() {
	reports, err := api.GetAllReports()
	if err != nil {
		log.Fatal(err)
	}
	for _, report := range reports {
		report.Delete()
	}
}

func TestGetReportsEmpty(t *testing.T) {
	dbClearReports()
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
			&lighthouse.LighthouseResult{Stdout: []byte("{}")},
			nil,
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
	dbClearReports()
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
	dbClearReports()
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
	dbClearReports()
}

func TestCreateReportFFMInvalid(t *testing.T) {
	body := []byte(`{"url": "https://www.google.com", "form_factor": "invalid"}`)
	response := createReport(t, body, false)
	checkResponseCode(t, http.StatusBadRequest, response)
	if body := response.Body.String(); strings.Contains(body, "Invalid form_factor") != true {
		t.Errorf("Expected body to contain Invalid form_factor. Got %s", body)
	}
	dbClearReports()
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

	dbClearReports()
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
