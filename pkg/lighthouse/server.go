package lighthouse

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
)

type Server struct {
	UnimplementedLighthouseServiceServer
	UseDocker bool
}

func (s *Server) Run(ctx context.Context, in *LighthouseRequest) (*LighthouseResult, error) {
	log.Printf("Received: %v", in.GetUrl())
	json, err := runLighthouse(in.GetUrl(), s.UseDocker)
	if err != nil {
		return nil, err
	} else {
		return &LighthouseResult{Stdout: json}, nil
	}
}

func runLighthouse(url string, useDocker bool) (json []byte, err error) {
	lhCommand := []string{}
	if useDocker {
		lhCommand = append(lhCommand, "docker", "run", "justinribeiro/lighthouse")
	}
	lhCommand = append(lhCommand, "lighthouse", "--chrome-flags=\"--no-sandbox --headless\"", url,
		"--output=json", "--output-path=stdout", "--emulated-form-factor=none")
	cmd := exec.Command(lhCommand[0], lhCommand[1:]...)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	log.Printf("Running command %+v", cmd)
	if err = cmd.Run(); err != nil {
		betterErr := fmt.Errorf("Error:%v, stderr: %s, stdout: %s", err, &stdErr, &stdOut)
		log.Println(betterErr)
		return nil, betterErr
	}
	return stdOut.Bytes(), nil
}
