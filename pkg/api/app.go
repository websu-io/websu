package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/swaggo/http-swagger"
	pb "github.com/websu-io/websu/pkg/lighthouse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type App struct {
	Router           *mux.Router
	LighthouseClient pb.LighthouseServiceClient
}

func ConnectToLighthouseServer(address string, secure bool) pb.LighthouseServiceClient {
	var opts []grpc.DialOption

	if secure {
		creds := credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: true,
		})
		opts = []grpc.DialOption{
			grpc.WithTransportCredentials(creds),
		}
	} else {
		opts = []grpc.DialOption{
			grpc.WithInsecure(),
		}
	}

	log.Printf("Connecting to gRPC Service [%s]", address)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Fatal(err)
	}
	return pb.NewLighthouseServiceClient(conn)
}

func NewApp() *App {
	a := new(App)
	a.SetupRoutes()
	return a
}

func (a *App) SetupRoutes() {
	a.Router = mux.NewRouter()
	a.Router.HandleFunc("/reports", a.getReports).Methods("GET")
	a.Router.HandleFunc("/reports", a.createReport).Methods("POST")
	a.Router.HandleFunc("/reports/{id}", a.getReport).Methods("GET")
	a.Router.HandleFunc("/reports/{id}", a.deleteReport).Methods("DELETE")
	a.Router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	spa := spaHandler{staticPath: "static", indexPath: "index.html"}
	a.Router.PathPrefix("/").Handler(spa)

}

func (a *App) Run(address string) {
	handler := cors.Default().Handler(a.Router)
	s := &http.Server{
		Addr:         address,
		Handler:      handler,
		ReadTimeout:  300 * time.Second,
		WriteTimeout: 300 * time.Second,
	}
	log.Printf("Listening on %s", address)
	log.Fatal(s.ListenAndServe())
}

func (a *App) getReports(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	reports, err := GetAllReports()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&reports)
}

// @Summary Create a Lighthouse Report
// @Description Run a lighthouse audit to generate a report. The field `raw_json` contains the
// @Description JSON output returned from lighthouse as a string. Note that `raw_json` field is
// @Description only returned during initial creation of the report.
// @Accept  json
// @Param ReportRequest body api.ReportRequest true "Lighthouse parameters to generate the report"
// @Produce  json
// @Success 200 {array} api.Report
// @Router /reports [post]
func (a *App) createReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var reportRequest ReportRequest
	if err := decodeJSONBody(w, r, &reportRequest); err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	log.Printf("Decoded json from HTTP body. ReportRequest: %+v", reportRequest)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*45)
	defer cancel()
	if reportRequest.FormFactor == "" {
		reportRequest.FormFactor = "desktop"
	}
	if reportRequest.FormFactor != "desktop" && reportRequest.FormFactor != "mobile" {
		err := errors.New("Invalid form_factor, must be desktop or mobile")
		log.Print(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if reportRequest.ThroughputKbps < 1000 {
		reportRequest.ThroughputKbps = 1000
	}
	lhOptions := []string{
		fmt.Sprintf("--emulated-form-factor=%v", reportRequest.FormFactor),
		fmt.Sprintf("--throttling.throughputKbps=%v", reportRequest.ThroughputKbps),
		"--throttling.rttMs=0",
		"--throttling.cpuSlowdownMultiplier=1",
		"--throttling.requestLatencyMs=0",
		"--throttling.downloadThroughputKbps=0",
		"--throttling.uploadThroughputKbps=0",
	}
	lhRequest := pb.LighthouseRequest{
		Url:     reportRequest.URL,
		Options: lhOptions,
	}
	lhResult, err := a.LighthouseClient.Run(ctx, &lhRequest)
	if err != nil {
		log.Printf("could not run lighthouse: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	report := NewReportFromRequest(&reportRequest)
	report.AuditResults, err = parseAuditResults(lhResult.GetStdout(), keys)
	if err != nil {
		log.Print("Error parsing audit results")
	}
	report.PerformanceScore = parsePerformanceScore(lhResult.GetStdout())
	report.RawJSON = string(lhResult.GetStdout())
	if err := report.Insert(); err != nil {
		log.Printf("unable to insert report: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&report)
}

func (a *App) getReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	report, err := GetReportByObjectIDHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&report)
}

func (a *App) deleteReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	report, err := GetReportByObjectIDHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := report.Delete(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&Report{})
}

// spaHandler implements the http.Handler interface, so we can use it
// to respond to HTTP requests. The path to the static directory and
// path to the index file within that static directory are used to
// serve the SPA in the given static directory.
type spaHandler struct {
	staticPath string
	indexPath  string
}

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// get the absolute path to prevent directory traversal
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		// if we failed to get the absolute path respond with a 400 bad request
		// and stop
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// prepend the path with the path to the static directory
	path = filepath.Join(h.staticPath, path)

	// check whether a file exists at the given path
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		// file does not exist, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static dir
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}
