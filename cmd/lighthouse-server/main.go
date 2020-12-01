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
	useDocker     = true
)

func main() {
	flag.StringVar(&listenAddress, "listen-address",
		cmd.GetenvString("LISTEN_ADDRESS", listenAddress),
		"The address and port to listen on. Default: \":50051\". Example with host: \"127.0.0.1:50051\"")
	flag.BoolVar(&useDocker, "use-docker",
		cmd.GetenvBool("USE_DOCKER", useDocker),
		"Boolean to indicate whether docker should be used to run lighthouse. Default: true. Possible values: true, false.")
	flag.Parse()

	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterLighthouseServiceServer(s, &pb.Server{UseDocker: useDocker})
	log.Printf("listening on %v", listenAddress)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
