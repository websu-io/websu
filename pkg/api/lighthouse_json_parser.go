package api

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

var keys = []string{
	"first-contentful-paint",
	"first-meaningful-paint",
	"speed-index",
	"estimated-input-latency",
	"total-blocking-time",
	"server-response-time",
}

type lhJsonResult struct {
	Audits map[string]json.RawMessage `json:"audits"`
}

func parseAuditResults(rawJson []byte, keys []string) (map[string]AuditResult, error) {
	res := lhJsonResult{}
	if err := json.Unmarshal(rawJson, &res); err != nil {
		return nil, err
	}
	auditResults := make(map[string]AuditResult)
	for _, key := range keys {
		ar, err := parseAuditResult(res.Audits[key])
		if err != nil {
			log.WithFields(log.Fields{
				"key":   key,
				"json":  string(res.Audits[key]),
				"error": err,
			}).Error("Error parsing audit result")
		} else {
			auditResults[key] = *ar
		}
	}
	return auditResults, nil
}

func parseAuditResult(rawJson []byte) (*AuditResult, error) {
	ar := &AuditResult{}
	if err := json.Unmarshal(rawJson, ar); err != nil {
		return nil, err
	}
	return ar, nil
}
