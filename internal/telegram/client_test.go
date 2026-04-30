package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestClientSendMessage(t *testing.T) {
	var got sendMessageRequest
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/botTOKEN/sendMessage" {
			t.Fatalf("path = %s, want /botTOKEN/sendMessage", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"ok":true}`)),
			Header:     make(http.Header),
		}, nil
	})}

	client := NewClientWithBaseURL("TOKEN", "https://api.telegram.test", httpClient)
	if err := client.SendMessage(context.Background(), "-100123", "<b>hello</b>"); err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}

	if got.ChatID != "-100123" {
		t.Fatalf("chat_id = %q, want -100123", got.ChatID)
	}
	if got.Text != "<b>hello</b>" {
		t.Fatalf("text = %q, want message", got.Text)
	}
	if got.ParseMode != "HTML" {
		t.Fatalf("parse_mode = %q, want HTML", got.ParseMode)
	}
	if !got.DisableWebPagePreview {
		t.Fatal("disable_web_page_preview = false, want true")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
