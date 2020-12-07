package api

import "testing"

func TestParseAuditresult(t *testing.T) {
	testString := `
{
    "id": "first-contentful-paint",
    "title": "First Contentful Paint",
    "description": "First Contentful Paint marks the time at which the first text or image is painted. [Learn more](https://web.dev/first-contentful-paint/).",
    "score": 0.38,
    "scoreDisplayMode": "numeric",
    "numericValue": 1822.8379999999997,
    "numericUnit": "millisecond",
    "displayValue": "1.8 s"
}
`
	got, err := parseAuditResult([]byte(testString))
	if err != nil {
		t.Errorf("Error %s parsing %s", err, testString)
	}
	if got == nil {
		t.Error("result is nil")
	}
	expected := AuditResult{ID: "first-contentful-paint"}
	if got.ID != "first-contentful-paint" {
		t.Errorf("got id: %s, but expected id: %s", got.ID, expected.ID)
	}
}

func TestParseAuditresults(t *testing.T) {
	testString := `
{
	"runWarnings": [],
	"audits": {
		"is-on-https": {
			"id": "is-on-https",
			"title": "Uses HTTPS",
			"description": "All sits.",
			"score": 1,
			"scoreDisplayMode": "binary",
			"displayValue": "",
			"details": {
				"type": "table",
				"headings": [],
				"items": []
			}
		},
		"first-contentful-paint": {
			"id": "first-contentful-paint",
			"title": "First Contentful Paint",
			"description": "First Contentful ",
			"score": 0.38,
			"scoreDisplayMode": "numeric",
			"numericValue": 1822.8379999999997,
			"numericUnit": "millisecond",
			"displayValue": "1.8 s"
		}
	}
}
`

	keys := []string{"first-contentful-paint"}
	got, err := parseAuditResults([]byte(testString), keys)
	if err != nil {
		t.Errorf("Error %s parsing %s", err, testString)
	}
	if got == nil {
		t.Error("result is nil")
	}
	if len(got) != 1 {
		t.Errorf("Expected only 1 AuditResult but got %v", len(got))
	}
}

func TestParsePerformanceScore(t *testing.T) {
	testString := `
{
  "categories": {
    "performance": {
      "title": "Performance",
      "id": "performance",
      "score": 0.65
    }
}
`
	got := parsePerformanceScore([]byte(testString))
	if got != float32(0.65) {
		t.Errorf("Expected 0.65 but got %v", got)
	}
}
