package main

import (
	"flag"
	"github.com/websu-io/websu/pkg/api"
	"github.com/websu-io/websu/pkg/cmd"
)

var (
	listenAddress    = ":8000"
	mongoURI         = "mongodb://localhost:27017"
	lighthouseServer = "localhost:50051"
)

func main() {
	flag.StringVar(&listenAddress, "listen-address",
		cmd.GetenvString("LISTEN_ADDRESS", listenAddress),
		"The address and port to listen on. Examples: \":8000\", \"127.0.0.1:8000\"")
	flag.StringVar(&mongoURI, "mongo-uri",
		cmd.GetenvString("MONGO_URI", mongoURI),
		"The MongoDB URI to connect to. Example: mongodb://localhost:27017")
	flag.StringVar(&lighthouseServer, "lighthouse-server",
		cmd.GetenvString("LIGHTHOUSE_SERVER", lighthouseServer),
		"The gRPC backend that runs lighthouse. Example: localhost:50051")
	flag.Parse()

	a := api.NewApp()
	a.LighthouseClient = api.ConnectToLighthouseServer(lighthouseServer)
	api.CreateMongoClient(mongoURI)
	a.Run(listenAddress)
}
