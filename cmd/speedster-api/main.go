package main

import (
	"github.com/reviewor-org/speedster/pkg/api"
	"os"
)

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	a := api.NewApp(mongoURI)
	a.Run(":8000")
}
