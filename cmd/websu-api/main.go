package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"github.com/websu-io/websu/docs"
	"github.com/websu-io/websu/pkg/api"
	"github.com/websu-io/websu/pkg/cmd"
)

var (
	listenAddress          = ":8000"
	mongoURI               = "mongodb://localhost:27017"
	lighthouseServer       = "localhost:50051"
	lighthouseServerSecure = false
	apiHost                = "localhost:8000"
	apiUrl                 = "http://localhost:8000"
	redisURL               = ""
	scheduler              = "go"
	gcpProject             = ""
	gcpRegion              = ""
	gcpTaskQueue           = ""
	serveFrontend          = true
	auth                   = ""
	smtpHost               = ""
	smtpPort               = 465
	smtpUsername           = ""
	smtpPassword           = ""
	fromEmail              = "info@websu.io"
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
	flag.StringVar(&apiHost, "api-host",
		cmd.GetenvString("API_HOST", apiHost),
		"The API hostname that's accessible from external users. Default: \"localhost:8000\", Example: \"websu.io\"")
	flag.StringVar(&apiUrl, "api-url",
		cmd.GetenvString("API_URL", apiUrl),
		"The API URL that's accessible from external users. Default: \"http://localhost:8000\", Example: \"https://websu.io\"")
	flag.StringVar(&mongoURI, "mongo-uri",
		cmd.GetenvString("MONGO_URI", mongoURI),
		"The MongoDB URI to connect to. Example: mongodb://localhost:27017")
	flag.StringVar(&lighthouseServer, "lighthouse-server",
		cmd.GetenvString("LIGHTHOUSE_SERVER", lighthouseServer),
		"The gRPC backend that runs lighthouse. Example: localhost:50051")
	flag.BoolVar(&lighthouseServerSecure, "lighthouse-server-secure",
		cmd.GetenvBool("LIGHTHOUSE_SERVER_SECURE", lighthouseServerSecure),
		"Boolean flag to indicate whether TLS should be used to connect to lighthouse server. Default: false")
	flag.StringVar(&redisURL, "redis-url",
		cmd.GetenvString("REDIS_URL", redisURL),
		`The Redis connection string to use. This setting is optional and by default
local memory will be used instead of Redis. Example redis://localhost:6379/0`)
	flag.StringVar(&scheduler, "scheduler",
		cmd.GetenvString("SCHEDULER", scheduler),
		"The scheduler to be used for running scheduled reports. This should be set to 'go' or 'gcp'. Default: 'go'")
	flag.StringVar(&gcpProject, "gcp-project",
		cmd.GetenvString("GCP_PROJECT", gcpProject),
		"The GCP project ID where the task queue is hosted. This setting is optional by default and only required if scheduler is set to GCP.")
	flag.StringVar(&gcpRegion, "gcp-region",
		cmd.GetenvString("GCP_REGION", gcpRegion),
		"The GCP region where the task queue is hosted. This setting is optional by default and only required if scheduler is set to GCP.")
	flag.StringVar(&gcpTaskQueue, "gcp-taskqueue",
		cmd.GetenvString("GCP_TASKQUEUE", gcpTaskQueue),
		"The GCP cloud task queue ID. This setting is optional by default and only required if scheduler is set to GCP.")
	flag.StringVar(&auth, "auth",
		cmd.GetenvString("AUTH", auth),
		"The authentication method to be used. This setting is optional and by default no authentication is done. Possible values: '' or 'firebase'.")
	flag.StringVar(&smtpHost, "smtp-host", cmd.GetenvString("SMTP_HOST", smtpHost),
		"SMTP host used for sending email. This setting is optional.")
	flag.IntVar(&smtpPort, "smtp-port", cmd.GetenvInt("SMTP_PORT", smtpPort),
		"SMTP port used for sending email. This setting is optional.")
	flag.StringVar(&smtpUsername, "smtp-username", cmd.GetenvString("SMTP_USERNAME", smtpUsername),
		"SMTP username used for sending email. This setting is optional.")
	flag.StringVar(&smtpPassword, "smtp-password", cmd.GetenvString("SMTP_PASSWORD", smtpPassword),
		"SMTP password used for sending email. This setting is optional.")
	flag.StringVar(&fromEmail, "from-email", cmd.GetenvString("FROM_EMAIL", fromEmail),
		"The email address of sender when sending email. This setting is optional.")
	flag.Parse()

	docs.SwaggerInfo.Host = apiHost
	options := make([]api.AppOption, 0)
	if redisURL != "" {
		options = append(options, api.WithRedis(redisURL))
	}
	api.ApiUrl = apiUrl
	api.Auth = auth
	if auth != "" && auth != "firebase" {
		log.Fatalf("--auth is currently set to %s, which isn't a valid value. Please use '' or 'firebase'", auth)
	}
	if auth == "firebase" {
		api.InitFirebase()
	}
	a := api.NewApp(options...)
	api.LighthouseClient = api.ConnectToLighthouseServer(lighthouseServer, lighthouseServerSecure)
	api.CreateMongoClient(mongoURI)
	api.ConnectLHLocations()
	api.Scheduler = scheduler
	if scheduler == "go" {
		s := api.GoScheduler{}
		s.Start()
	}
	if scheduler == "gcp" && (gcpProject == "" || gcpRegion == "" || gcpTaskQueue == "") {
		log.Fatal("Please set the GCP project, region and task queue id when scheduler is set to GCP. See -h for more information.")
	}
	if scheduler == "gcp" {
		api.GCPProject = gcpProject
		api.GCPRegion = gcpRegion
		api.GCPTaskQueue = gcpTaskQueue
	}

	api.SmtpHost = smtpHost
	api.SmtpPort = smtpPort
	api.SmtpUsername = smtpUsername
	api.SmtpPassword = smtpPassword
	api.FromEmail = fromEmail

	a.Run(listenAddress)
}
