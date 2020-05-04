package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type App struct {
	Router *mux.Router
}

// "mongodb://localhost:27017"
func NewApp() *App {
	a := new(App)
	a.SetupRoutes()
	return a
}

func (a *App) SetupRoutes() {
	a.Router = mux.NewRouter()
	a.Router.HandleFunc("/scans", a.getScans).Methods("GET")
	a.Router.HandleFunc("/scans", a.createScan).Methods("POST")
	a.Router.HandleFunc("/scans/{id}", a.getScan).Methods("GET")
	a.Router.HandleFunc("/scans/{id}", a.deleteScan).Methods("DELETE")
}

func (a *App) Run(address string) {
	log.Print("Listening on :8000")
	http.ListenAndServe(address, a.Router)
}

func (a *App) getScans(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	scans, err := GetAllScans()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&scans)
}

func (a *App) createScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var scan Scan
	err := decodeJSONBody(w, r, &scan)
	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	scan.ID = primitive.NewObjectID()
	scan.CreatedAt = time.Now()
	log.Printf("Decoded json from HTTP body. Scan: %+v", scan)

	html, jsonResult := runLightHouse(scan.URL, "/home/chrome/reports/speedster")
	scan.HTML = string(html)
	scan.JSON = string(jsonResult)
	if err := scan.Insert(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	json.NewEncoder(w).Encode(&scan)
}

func (a *App) getScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	scan, err := GetScanByObjectIDHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&scan)
}

func (a *App) deleteScan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	if err := DeleteScanByObjectIDHex(params["id"]); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&Scan{})
}

func runLightHouse(url, outputPath string) (html, json []byte) {
	// lighthouse --chrome-flags="--headless" $URL --output="html" --output=json --output-path=/tmp/$URL
	cmd := exec.Command("lighthouse", "--chrome-flags=\"--headless\"", url,
		"--output=json", "--output=html", "--output-path="+outputPath)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	log.Printf("Running command %+v", cmd)

	if err := cmd.Run(); err != nil {
		log.Print(err)
		return nil, nil
	}
	defer os.Remove(outputPath + ".report.json")
	defer os.Remove(outputPath + ".report.html")

	var err error
	json, err = ioutil.ReadFile(outputPath + ".report.json")
	if err != nil {
		log.Print("Error reading lighthouse json output file:", err)
		return nil, nil
	}
	html, err = ioutil.ReadFile(outputPath + ".report.html")
	if err != nil {
		log.Print("Error reading lighthouse html output file:", err)
		return nil, nil
	}

	return html, json
}
