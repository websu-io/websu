package main

import (
	"flag"
	"github.com/websu-io/websu/pkg/cmd"
	pb "github.com/websu-io/websu/pkg/lighthouse"
	"google.golang.org/grpc"
	"log"
	"net"
)

var (
	listenAddress = ":50051"
)

func main() {
	flag.StringVar(&listenAddress, "listen-address",
		cmd.Getenv("LISTEN_ADDRESS", listenAddress),
		"The address and port to listen on. Examples: \":50051\", \"127.0.0.1:50051\"")

	flag.Parse()

	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterLighthouseServiceServer(s, &pb.Server{})
	log.Printf("listening on %v", listenAddress)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
