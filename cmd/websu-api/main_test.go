package main

import (
	"bytes"
	"encoding/json"
	"github.com/websu-io/websu/pkg/api"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var a *api.App

func TestMain(m *testing.M) {
	a = api.NewApp()
	api.CreateMongoClient("mongodb://localhost:27017")
	code := m.Run()
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

func dbClearScans() {
	scans, err := api.GetAllScans()
	if err != nil {
		log.Fatal(err)
	}
	for _, scan := range scans {
		scan.Delete()
	}
}

func TestGetScansEmpty(t *testing.T) {
	dbClearScans()
	req, _ := http.NewRequest("GET", "/scans", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
		t.Errorf("Expected an empty array as []. Got %s", body)
	}
}

func createScan() *httptest.ResponseRecorder {
	scan := bytes.NewBuffer([]byte(`{"URL": "https://reviewor.org"}`))
	req, _ := http.NewRequest("POST", "/scans", scan)
	return executeRequest(req)
}

func TestCreateScan(t *testing.T) {
	response := createScan()
	checkResponseCode(t, http.StatusOK, response)
	if body := response.Body.String(); strings.Contains(body, "reviewor.org") != true {
		t.Errorf("Expected body to contain reviewor.org. Got %s", body)
	}
	dbClearScans()
}

func TestCreateGetandDeleteScan(t *testing.T) {
	r := createScan()
	checkResponseCode(t, http.StatusOK, r)

	var scan api.Scan
	if err := json.NewDecoder(r.Body).Decode(&scan); err != nil {
		t.Errorf("Error: %s. Json decoding body: %s\n", err, r.Body)
	}

	req, _ := http.NewRequest("GET", "/scans/"+scan.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)

	req, _ = http.NewRequest("DELETE", "/scans/"+scan.ID.Hex(), nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusOK, r)

	dbClearScans()
}

func TestDeleteScanNonExisting(t *testing.T) {
	// not a valid hex string
	req, _ := http.NewRequest("DELETE", "/scans/doesnotexist", nil)
	log.Printf("Request: %+v", req)
	r := executeRequest(req)
	checkResponseCode(t, http.StatusBadRequest, r)
	log.Printf("Response: %+v", r)

	// valid hex string but doesnt exist
	req, _ = http.NewRequest("DELETE", "/scans/5eab5a25b830c33d857dc045", nil)
	log.Printf("Request: %+v", req)
	r = executeRequest(req)
	checkResponseCode(t, http.StatusBadRequest, r)
	log.Printf("Response: %+v", r)

}
