package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
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
	a.Router.HandleFunc("/scans", a.getScans).Methods("GET")
	a.Router.HandleFunc("/scans", a.createScan).Methods("POST")
	a.Router.HandleFunc("/scans/{id}", a.getScan).Methods("GET")
	a.Router.HandleFunc("/scans/{id}", a.deleteScan).Methods("DELETE")
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
	if err := decodeJSONBody(w, r, &scan); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*45)
	defer cancel()
	lhResult, err := a.LighthouseClient.Run(ctx, &pb.LighthouseRequest{Url: scan.URL})
	if err != nil {
		log.Printf("could not run lighthouse: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	scan.Json = string(lhResult.GetStdout())
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
	scan, err := GetScanByObjectIDHex(params["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := scan.Delete(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(&Scan{})
}
