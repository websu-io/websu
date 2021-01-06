package api

import (
	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2"
)

type GCPScheduler struct {
	Project  string
	Location string
	Queue    string
}

func (g GCPScheduler) RunReport(sr ScheduledReport) {
	jsonBody, err := json.Marshal(sr.ReportRequest)
	if err != nil {
		log.WithError(err).WithField("ScheduledReport", sr).Error("Unable to marshal http RunReport request")
		return
	}
	_, err = CreateGCPCloudTask(g.Project, g.Location, g.Queue, ApiUrl+"/reports", jsonBody)
	if err != nil {
		log.WithError(err).WithField("ScheduledReport", sr).Error("Unable to create GCP cloud task")
		return
	}

}

func CreateGCPCloudTask(projectID, locationID, queueID, url string, body []byte) (*taskspb.Task, error) {
	// Create a new Cloud Tasks client instance.
	// See https://godoc.org/cloud.google.com/go/cloudtasks/apiv2
	ctx := context.Background()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewClient: %v", err)
	}

	// Build the Task queue path.
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2#HttpRequest
			MessageType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        url,
				},
			},
		},
	}

	// Add a payload message if one is present.
	req.Task.GetHttpRequest().Body = body

	createdTask, err := client.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks.CreateTask: %v", err)
	}
	return createdTask, nil
}
