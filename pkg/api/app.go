package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	libredis "github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"github.com/swaggo/http-swagger"
	mhttp "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	pb "github.com/websu-io/websu/pkg/lighthouse"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net/http"
	"runtime/debug"
	"time"
)

var (
	DefaultRateLimit  = "10-M"
	LighthouseClient  pb.LighthouseServiceClient
	LighthouseClients map[string]pb.LighthouseServiceClient
)

type App struct {
	Router      *mux.Router
	RedisClient *libredis.Client
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

	log.Infof("Connecting to gRPC Service [%s]", address)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		log.Error(err)
	}
	return pb.NewLighthouseServiceClient(conn)
}

func ConnectLHLocations() {
	locations, err := GetAllLocations()
	if err != nil {
		log.Errorf("Error getting locations while trying to connect: %v", err)
	} else {
		for _, location := range locations {
			LighthouseClients[location.Name] = ConnectToLighthouseServer(location.Address, location.Secure)
		}
	}
}

type AppOption func(c *App)

func WithRedis(redisURL string) AppOption {
	return func(a *App) {
		a.RedisClient = CreateRedisClient(redisURL)
	}
}

func NewApp(opts ...AppOption) *App {
	a := new(App)
	for _, opt := range opts {
		opt(a)
	}
	a.SetupRoutes()
	LighthouseClients = make(map[string]pb.LighthouseServiceClient)
	return a
}

func (a *App) SetupRoutes() {
	var limiter *mhttp.Middleware
	if a.RedisClient != nil {
		log.Info("Using redis based rate limiter")
		limiter = CreateRedisRateLimiter(DefaultRateLimit, "default-limiter", a.RedisClient)
	} else {
		log.Info("Using memory based rate limiter")
		limiter = CreateMemRateLimiter(DefaultRateLimit)
	}
	a.Router = mux.NewRouter()
	a.Router.HandleFunc("/reports", a.getReports).Methods("GET")
	a.Router.Handle("/reports", limiter.Handler(http.HandlerFunc(a.createReport))).Methods("POST")
	a.Router.HandleFunc("/reports/{id}", a.getReport).Methods("GET")
	a.Router.HandleFunc("/reports/{id}", a.deleteReport).Methods("DELETE")
	a.Router.PathPrefix("/docs/").Handler(httpSwagger.WrapHandler)
	a.Router.HandleFunc("/locations", a.getLocations).Methods("GET")
	a.Router.HandleFunc("/locations", a.createLocation).Methods("POST")
	a.Router.HandleFunc("/locations/{id}", a.deleteLocation).Methods("DELETE")
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
	log.Infof("Listening on %s", address)
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
// @Description JSON output returned from lighthouse as a string.
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
			log.WithError(err).Error("Malformed Request during decoding JSON of createReport")
			http.Error(w, mr.msg, mr.status)
		} else {
			log.WithError(err).Error("Error decoding JSON")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	log.Infof("Decoded json from HTTP body. ReportRequest: %+v", reportRequest)
	if err := reportRequest.Validate(); err != nil {
		log.WithError(err).WithField("reportRequest", reportRequest).Info("Unable to validate ReportRequest")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if reportRequest.FormFactor == "" {
		reportRequest.FormFactor = "desktop"
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
	var lhClient pb.LighthouseServiceClient
	if val, ok := LighthouseClients[reportRequest.Location]; ok {
		lhClient = val
	} else {
		lhClient = LighthouseClient
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*45)
	defer cancel()
	lhResult, err := lhClient.Run(ctx, &lhRequest)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"lhRequest": fmt.Sprintf("%+v", lhRequest),
		}).Error("Could not run lighthouse\n", string(debug.Stack()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	report := NewReportFromRequest(&reportRequest)
	report.AuditResults, err = parseAuditResults(lhResult.GetStdout(), keys)
	if err != nil {
		log.WithError(err).Error("Error parsing audit results")
	}
	report.PerformanceScore = parsePerformanceScore(lhResult.GetStdout())
	report.RawJSON = string(lhResult.GetStdout())
	if err := report.Insert(); err != nil {
		log.WithError(err).Error("unable to insert report")
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

func (a *App) getLocations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	locations, err := GetAllLocations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&locations)
}

func (a *App) createLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	location := NewLocation()
	if err := decodeJSONBody(w, r, &location); err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.WithError(err).Error("Error decoding location json")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	log.Infof("Decoded json from HTTP body. Location: %+v", location)
	if err := location.Insert(); err != nil {
		log.WithError(err).Error("Error creating location")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	ConnectLHLocations()
	json.NewEncoder(w).Encode(&location)
}

func (a *App) deleteLocation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	location, err := GetLocationByObjectIDHex(params["id"])
	log.WithFields(log.Fields{
		"location": location,
		"params":   params,
	}).Info("Deleting location")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := location.Delete(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&Report{})
}
