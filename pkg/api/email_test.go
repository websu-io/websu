package api

import (
	"net/smtp"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
)

// Needed to load relative path to templates/email-template.html
func init() {
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../..")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

type emailRecorder struct {
	Addr string
	Auth smtp.Auth
	From string
	To   []string
	Msg  []byte
}

func TestSendEmail(t *testing.T) {
	actual := new(emailRecorder)
	sendEmail = func(server string, auth smtp.Auth, fromEmail string, to []string, content []byte) error {
		*actual = emailRecorder{server, auth, fromEmail, to, content}
		return nil
	}
	r := NewReport()
	r.URL = "https://www.google.com"
	r.Email = "test@websu.io"
	r.Location = "us-central1"
	r.PerformanceScore = 888
	err := r.SendEmail()
	if err != nil {
		t.Error(err.Error())
	}
	if strings.Contains(string(actual.Msg), r.URL) != true {
		t.Errorf("Expected email msg to contain URL %s", r.URL)
	}
	if strings.Contains(string(actual.Msg), "888") != true {
		t.Error("Expected email msg to contain 888, which was the performance score")
	}
	if strings.Contains(string(actual.Msg), r.Location) != true {
		t.Errorf("Expected email msg to contain location %s", r.Location)
	}
	if strings.Contains(string(actual.Msg), "ObjectID") == true {
		t.Error("ObjectID shouldn't be part of the message")
	}

}
