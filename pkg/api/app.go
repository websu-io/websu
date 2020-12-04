package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/swaggo/http-swagger"
	pb "github.com/websu-io/websu/pkg/lighthouse"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
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
	a.Router.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

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
// @Param body body api.ReportInput true "Lighthouse parameters to generate the report"
// @Produce  json
// @Success 200 {array} api.Report
// @Router /reports [post]
func (a *App) createReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var report Report
	if err := decodeJSONBody(w, r, &report); err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	report.ID = primitive.NewObjectID()
	report.CreatedAt = time.Now()
	log.Printf("Decoded json from HTTP body. Report: %+v", report)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*45)
	defer cancel()
	lhResult, err := a.LighthouseClient.Run(ctx, &pb.LighthouseRequest{Url: report.URL})
	if err != nil {
		log.Printf("could not run lighthouse: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
