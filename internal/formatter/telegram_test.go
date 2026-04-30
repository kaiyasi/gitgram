package formatter

import (
	"strings"
	"testing"

	"github.com/yorukot/gitgram/internal/activity"
)

func TestTelegramHTMLEscapesUserContent(t *testing.T) {
	msg := TelegramHTML(activity.Activity{
		Event:   activity.EventIssueComment,
		Repo:    "owner/repo",
		Action:  "created",
		Subject: "issue",
		Number:  42,
		Title:   `Fix <login> & "oauth"`,
		Actor:   "octocat",
		Summary: `<b>do not parse me</b>`,
		URL:     `https://github.com/owner/repo/issues/42?x=1&y=2`,
	})

	for _, want := range []string{
		"Fix &lt;login&gt; &amp; &#34;oauth&#34;",
		"&lt;b&gt;do not parse me&lt;/b&gt;",
		`href="https://github.com/owner/repo/issues/42?x=1&amp;y=2"`,
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message does not contain %q:\n%s", want, msg)
		}
	}
}

func TestTelegramHTMLWorkflowFailure(t *testing.T) {
	msg := TelegramHTML(activity.Activity{
		Event:  activity.EventWorkflowRun,
		Repo:   "owner/repo",
		Action: "failed",
		Title:  "CI",
		Branch: "main",
		Actor:  "octocat",
		URL:    "https://github.com/owner/repo/actions/runs/1",
	})

	for _, want := range []string{
		"<b>owner/repo</b> workflow failed",
		"CI on <code>main</code>",
		"by <code>octocat</code>",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message does not contain %q:\n%s", want, msg)
		}
	}
}
