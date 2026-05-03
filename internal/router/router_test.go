package router_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yorukot/gitgram/internal/router"
	"github.com/yorukot/gitgram/internal/store"
)

func TestGitHubWebhookHandlerSendsAndDedupes(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{
		"action": "opened",
		"number": 12,
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"pull_request": {
			"html_url": "https://github.com/owner/repo/pull/12",
			"title": "Add login",
			"number": 12,
			"merged": false,
			"user": {"login": "octocat"},
			"head": {"ref": "feature/login"},
			"base": {"ref": "main"}
		}
	}`)

	sender := &fakeSender{}
	handler := router.GitHubWebhookHandler{
		Secret: secret,
		ChatID: "-100123",
		AllowedRepo: func(repo string) bool {
			return repo == "owner/repo"
		},
		Store:  store.NewMemoryDeliveryStore(100),
		Sender: sender,
	}

	for i := 0; i < 2; i++ {
		req := signedWebhookRequest(secret, "pull_request", "delivery-1", body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d body = %s", i+1, rec.Code, rec.Body.String())
		}
	}

	if sender.calls != 1 {
		t.Fatalf("sender calls = %d, want 1", sender.calls)
	}
	if sender.chatID != "-100123" {
		t.Fatalf("chatID = %q, want -100123", sender.chatID)
	}
	if !strings.Contains(sender.text, "PR opened") {
		t.Fatalf("message does not look like pull request notification:\n%s", sender.text)
	}
}

func TestGitHubWebhookHandlerIgnoresPullRequestUpdatesByDefault(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{
		"action": "synchronize",
		"number": 12,
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"pull_request": {
			"html_url": "https://github.com/owner/repo/pull/12",
			"title": "Add login",
			"number": 12,
			"merged": false,
			"user": {"login": "octocat"},
			"head": {"ref": "feature/login"},
			"base": {"ref": "main"}
		}
	}`)

	sender := &fakeSender{}
	handler := router.GitHubWebhookHandler{
		Secret:      secret,
		ChatID:      "-100123",
		AllowedRepo: func(repo string) bool { return repo == "owner/repo" },
		Store:       store.NewMemoryDeliveryStore(100),
		Sender:      sender,
	}

	req := signedWebhookRequest(secret, "pull_request", "delivery-pr-update", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls = %d, want 0", sender.calls)
	}
}

func TestGitHubWebhookHandlerFiltersPushBranches(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{
		"ref": "refs/heads/feature/login",
		"compare": "https://github.com/owner/repo/compare/a...b",
		"repository": {"full_name": "owner/repo"},
		"sender": {"login": "octocat"},
		"commits": [
			{
				"id": "abcdef1234567890",
				"message": "Fix login",
				"url": "https://github.com/owner/repo/commit/abcdef",
				"author": {"name": "Mona"}
			}
		]
	}`)

	sender := &fakeSender{}
	handler := router.GitHubWebhookHandler{
		Secret:          secret,
		ChatID:          "-100123",
		AllowedRepo:     func(repo string) bool { return repo == "owner/repo" },
		ImportantBranch: func(branch string) bool { return branch == "main" },
		Store:           store.NewMemoryDeliveryStore(100),
		Sender:          sender,
	}

	req := signedWebhookRequest(secret, "push", "delivery-push-feature", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls = %d, want 0", sender.calls)
	}
}

func TestGitHubWebhookHandlerIgnoresSuccessfulWorkflowRun(t *testing.T) {
	secret := "test-secret"
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

	sender := &fakeSender{}
	handler := router.GitHubWebhookHandler{
		Secret:      secret,
		ChatID:      "-100123",
		AllowedRepo: func(repo string) bool { return repo == "owner/repo" },
		Store:       store.NewMemoryDeliveryStore(100),
		Sender:      sender,
	}

	req := signedWebhookRequest(secret, "workflow_run", "delivery-2", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls = %d, want 0", sender.calls)
	}
}

type fakeSender struct {
	calls  int
	chatID string
	text   string
}

func (s *fakeSender) SendMessage(_ context.Context, chatID string, text string) error {
	s.calls++
	s.chatID = chatID
	s.text = text
	return nil
}

func signedWebhookRequest(secret, event, deliveryID string, body []byte) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-GitHub-Delivery", deliveryID)
	req.Header.Set("X-Hub-Signature-256", signWebhookBody(secret, body))
	return req
}

func signWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
