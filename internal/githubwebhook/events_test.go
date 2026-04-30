package githubwebhook

import (
	"errors"
	"testing"

	"github.com/yorukot/gitgram/internal/activity"
)

func TestParseWorkflowRunIgnoresSuccess(t *testing.T) {
	body := []byte(`{
		"action": "completed",
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"workflow_run": {
			"name": "CI",
			"head_branch": "main",
			"conclusion": "success",
			"html_url": "https://github.com/owner/repo/actions/runs/1"
		}
	}`)

	_, err := ParseEvent(activity.EventWorkflowRun, "delivery-1", body)
	if !errors.Is(err, ErrIgnored) {
		t.Fatalf("expected ErrIgnored, got %v", err)
	}
}

func TestParseWorkflowRunFailure(t *testing.T) {
	body := []byte(`{
		"action": "completed",
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"workflow_run": {
			"name": "CI",
			"head_branch": "main",
			"conclusion": "failure",
			"html_url": "https://github.com/owner/repo/actions/runs/2"
		}
	}`)

	got, err := ParseEvent(activity.EventWorkflowRun, "delivery-2", body)
	if err != nil {
		t.Fatalf("ParseEvent returned error: %v", err)
	}
	if got.Event != activity.EventWorkflowRun {
		t.Fatalf("event = %q, want %q", got.Event, activity.EventWorkflowRun)
	}
	if got.Action != "failed" {
		t.Fatalf("action = %q, want failed", got.Action)
	}
	if got.Repo != "owner/repo" {
		t.Fatalf("repo = %q, want owner/repo", got.Repo)
	}
	if got.Branch != "main" {
		t.Fatalf("branch = %q, want main", got.Branch)
	}
}

func TestParsePush(t *testing.T) {
	body := []byte(`{
		"ref": "refs/heads/main",
		"compare": "https://github.com/owner/repo/compare/a...b",
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"commits": [
			{
				"id": "abcdef1234567890",
				"message": "Fix login\n\nLong body",
				"url": "https://github.com/owner/repo/commit/abcdef",
				"author": {"name": "Mona"}
			}
		]
	}`)

	got, err := ParseEvent(activity.EventPush, "delivery-3", body)
	if err != nil {
		t.Fatalf("ParseEvent returned error: %v", err)
	}
	if got.Branch != "main" {
		t.Fatalf("branch = %q, want main", got.Branch)
	}
	if len(got.Commits) != 1 {
		t.Fatalf("len(commits) = %d, want 1", len(got.Commits))
	}
	if got.Commits[0].Message != "Fix login" {
		t.Fatalf("message = %q, want first line", got.Commits[0].Message)
	}
}
