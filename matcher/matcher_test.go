package matcher

import (
	"encoding/json"
	"testing"
)

func TestSuccessfulMatcher(t *testing.T) {
	payload := map[string]any{
		"status": "completed",
		"workflow_run": map[string]any{
			"name": "Parsa",
			"role": "admin",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	condition := map[string]string{
		"status":            "completed",
		"workflow_run.name": "Parsa",
		"workflow_run.role": "admin",
	}
	yes, err := MatchesPayload(body, condition)

	if err != nil {
		t.Errorf("Expeceted nil, got an error: %s", err)
	}

	if !yes {
		t.Error("Expected to match.")
	}
}

func TestFailureMatcher(t *testing.T) {
	body := []byte(`{
		"status": "completed",
		"workflow_run": {
			"name": "Parsa",
			"role": "admin"
		}
	}`)

	condition := map[string]string{
		"status":                  "completed",
		"workflow_run.conclusion": "Ahmad",
		"workflow_run.role":       "Saman",
	}
	yes, err := MatchesPayload(body, condition)

	if err != nil {
		t.Errorf("Unexpected error, %s", err)
	}

	if yes {
		t.Error("Expected to match error.")
	}

	// A broken body
	body = []byte(`{
		"status",
		"workflow_run": {
			"name": "Parsa",
			"role": "admin"
		}
	}`)

	_, err = MatchesPayload(body, condition)

	if err == nil {
		t.Error("Expected an error. Got nil.")
	}

}
