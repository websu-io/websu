package api

import (
	"bytes"
	"encoding/json"
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

var (
	httpClient *http.Client
	timeout    = 60
)

func init() {
	httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: time.Second * 60,
	}
}

type ReportRunner interface {
	RunReport(sr ScheduledReport)
}

func HTTPRunReport(r ReportRequest) {
	jsonBody, err := json.Marshal(r)
	if err != nil {
		log.WithError(err).WithField("r", r).Error("Unable to marshal http RunReport request")
		return
	}
	req, err := http.NewRequest("POST", ApiUrl+"/reports", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.WithError(err).WithField("req", req).Error("Unable to create http RunReport request")
		return
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithError(err).WithField("req", req).WithField("resp", resp).Error("Unable to execute scheduled RunReport request")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		log.WithField("reportRequest", r).Info("The scheduled report was created successfully")
	} else {
		log.WithFields(log.Fields{
			"reportRequest": r,
			"response":      resp,
		}).Error("Scheduled Run HTTP response code seems like an error")
	}
}

type GoScheduler struct{}

func (gs *GoScheduler) Start() {
	s := gocron.NewScheduler(time.UTC)
	s.Every(10).Minutes().Do(RunScheduledReports, gs)
	s.StartAsync()
}

func (g GoScheduler) RunReport(sr ScheduledReport) {
	go HTTPRunReport(sr.ReportRequest)
}

func RunScheduledReports(reportRunner ReportRunner) int {
	log.Info("Running scheduled reports")
	scheduledReports, err := GetScheduleReportsDueToRun()
	if err != nil {
		log.WithError(err).Error("Error when getting reports due from database")
		return 0
	}
	for _, sr := range scheduledReports {
		log.WithField("ScheduledReport", sr).Info("Running scheduled report")
		reportRunner.RunReport(sr)
		sr.LastRun = time.Now()
		sr.Update()
	}
	return len(scheduledReports)
}
