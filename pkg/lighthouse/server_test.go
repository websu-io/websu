package lighthouse

import (
	"os/exec"
	"strconv"
	"testing"
)

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func TestRunLighthouse(t *testing.T) {
	if !commandExists("lighthouse") && !commandExists("docker") {
		t.Skip("lighthouse executable not available so skipping the test")
		return
	}
	tests := []bool{}
	if commandExists("lighthouse") {
		tests = append(tests, false)
	}
	if commandExists("docker") {
		tests = append(tests, true)
	}

	for _, useDocker := range tests {
		useDocker := useDocker // capture range variable
		t.Run("TestRunLighthouse-useDocker-"+strconv.FormatBool(useDocker), func(t *testing.T) {
			t.Parallel()
			options := []string{}
			chromeflags := []string{}
			jsonResult, err := runLighthouse("https://www.samos-it.com", useDocker, options, chromeflags)
			if err != nil {
				t.Errorf("Error running lighthouse: %v\n", err)
			}
			if len(jsonResult) < 5 {
				t.Error("Expecting a json result that's bigger than 4")
			}

		})
	}

}
