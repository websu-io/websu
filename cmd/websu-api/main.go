package main

import (
	"flag"
	_ "github.com/websu-io/websu/docs"
	"github.com/websu-io/websu/pkg/api"
	"github.com/websu-io/websu/pkg/cmd"
)

var (
	listenAddress          = ":8000"
	mongoURI               = "mongodb://localhost:27017"
	lighthouseServer       = "localhost:50051"
	lighthouseServerSecure = false
)

// @title Websu API
// @version 1.0
// @description Run lighthouse as a service
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
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
	flag.BoolVar(&lighthouseServerSecure, "lighthouse-server-secure",
		cmd.GetenvBool("LIGHTHOUSE_SERVER_SECURE", lighthouseServerSecure),
		"Boolean flag to indicate whether TLS should be used to connect to lighthouse server. Default: false")
	flag.Parse()

	a := api.NewApp()
	a.LighthouseClient = api.ConnectToLighthouseServer(lighthouseServer, lighthouseServerSecure)
	api.CreateMongoClient(mongoURI)
	a.Run(listenAddress)
}
