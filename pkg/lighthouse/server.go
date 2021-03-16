package lighthouse

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Server struct {
	UnimplementedLighthouseServiceServer
	UseDocker bool
}

func (s *Server) Run(ctx context.Context, in *LighthouseRequest) (*LighthouseResult, error) {
	log.Printf("Received: %v", in.GetUrl())
	json, err := runLighthouse(in.GetUrl(), s.UseDocker, in.GetOptions(), in.GetChromeflags())
	if err != nil {
		return nil, err
	} else {
		return &LighthouseResult{Stdout: json}, nil
	}
}

func runLighthouse(url string, useDocker bool, options []string, chromeflags []string) (json []byte, err error) {
	lhCommand := []string{}
	if useDocker {
		lhCommand = append(lhCommand, "docker", "run", "samos123/lighthouse")
	}
	defaultChromeflags := []string{"--no-sandbox", "--headless"}
	chromeflags = append(defaultChromeflags, chromeflags...)
	lhCommand = append(lhCommand, "lighthouse", url,
		fmt.Sprintf("--chrome-flags=\"%s\"", strings.Join(chromeflags, " ")),
		"--output=json", "--output-path=stdout", "--disable-dev-shm-usage")
	lhCommand = append(lhCommand, options...)

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
