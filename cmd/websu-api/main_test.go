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

func createReport(t *testing.T) *httptest.ResponseRecorder {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockLightHouseClient := mocks.NewMockLighthouseServiceClient(ctrl)
	a.LighthouseClient = mockLightHouseClient
	report := bytes.NewBuffer([]byte(`{"URL": "https://reviewor.org"}`))
	req, _ := http.NewRequest("POST", "/reports", report)
	mockLightHouseClient.EXPECT().Run(gomock.Any(), gomock.Any()).Return(
		&lighthouse.LighthouseResult{Stdout: []byte("{}")},
		nil,
	)
	resp := executeRequest(req)
	return resp
}

func TestCreateReport(t *testing.T) {
	response := createReport(t)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "reviewor.org") != true {
		t.Errorf("Expected body to contain reviewor.org. Got %s", body)
	}
	dbClearReports()
}

func TestCreateGetandDeleteReport(t *testing.T) {
	r := createReport(t)
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
