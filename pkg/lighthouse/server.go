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
		lhCommand = append(lhCommand, "docker", "run", "samos123/lighthouse:9.4.0")
	}
	defaultChromeflags := []string{"--no-sandbox", "--headless", "--disable-dev-shm-usage",
		"--hide-scrollbars", "--disable-features=TranslateUI", "--disable-extensions",
		"--disable-component-extensions-with-background-pages", "--disable-background-networking", "--disable-sync",
		"--metrics-recording-only", "--disable-default-apps", "--mute-audio", "--no-default-browser-check",
		"--no-first-run", "--disable-backgrounding-occluded-windows", "--disable-renderer-backgrounding",
		"--disable-background-timer-throttling", "--force-fieldtrials=*BackgroundTracing/default/",
		"--use-gl=swiftshader", "--disable-software-rasterizer"}
	chromeflags = append(defaultChromeflags, chromeflags...)
	lhCommand = append(lhCommand, "lighthouse", url,
		fmt.Sprintf("--chrome-flags=\"%s\"", strings.Join(chromeflags, " ")),
		"--output=json", "--output-path=stdout", "--disable-dev-shm-usage",
		"--only-categories=best-practices,performance,seo",
		"--skip-audits=final-screenshot,screenshot-thumbnails,apple-touch-icon")

	// Update deprecated options that were in lighthouse 6.4
	for i, option := range options {
		if strings.HasPrefix(option, "--emulated-form-factor") {
			formFactor := strings.Split(option, "=")[1]
			options[i] = "--form-factor=" + formFactor
			if formFactor == "desktop" {
				options = append(options, "--screenEmulation.mobile=false")
				options = append(options, "--screenEmulation.height=940")
				options = append(options, "--screenEmulation.width=1350")
				options = append(options, "--screenEmulation.deviceScaleFactor=1")
				options = append(options, "--emulatedUserAgent=Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4695.0 Safari/537.36 Chrome-Lighthouse")
			}
			if formFactor == "mobile" {
				options = append(options, "--screenEmulation.mobile=true")
				options = append(options, "--screenEmulation.height=640")
				options = append(options, "--screenEmulation.width=360")
				options = append(options, "--screenEmulation.deviceScaleFactor=2.625")
				options = append(options, "--emulatedUserAgent=Mozilla/5.0 (Linux; Android 7.0; Moto G (4)) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4695.0 Mobile Safari/537.36 Chrome-Lighthouse")
			}

			break
		}
	}
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
