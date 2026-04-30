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

func TestTelegramHTMLTruncatesOnlyPullRequestCommentBody(t *testing.T) {
	longText := strings.Repeat("a", maxCommentRunes+20)

	commentMsg := TelegramHTML(activity.Activity{
		Event:   activity.EventIssueComment,
		Repo:    "owner/repo",
		Action:  "created",
		Subject: "pull request",
		Number:  12,
		Title:   "Add login",
		Actor:   "octocat",
		Summary: longText,
		URL:     "https://github.com/owner/repo/pull/12#issuecomment-1",
	})

	if !strings.Contains(commentMsg, "Comment truncated. Open on GitHub for full text.") {
		t.Fatalf("comment message should include truncation notice:\n%s", commentMsg)
	}
	if strings.Contains(commentMsg, strings.Repeat("a", maxCommentRunes+1)) {
		t.Fatalf("comment message should be truncated:\n%s", commentMsg)
	}

	issueCommentMsg := TelegramHTML(activity.Activity{
		Event:   activity.EventIssueComment,
		Repo:    "owner/repo",
		Action:  "created",
		Subject: "issue",
		Number:  42,
		Title:   "Cannot login",
		Actor:   "octocat",
		Summary: longText,
		URL:     "https://github.com/owner/repo/issues/42#issuecomment-1",
	})

	if strings.Contains(issueCommentMsg, "Comment truncated. Open on GitHub for full text.") {
		t.Fatalf("issue comment should not include PR comment truncation notice:\n%s", issueCommentMsg)
	}
	if !strings.Contains(issueCommentMsg, strings.Repeat("a", maxCommentRunes+20)) {
		t.Fatalf("issue comment should not use PR comment-specific truncation:\n%s", issueCommentMsg)
	}

	reviewMsg := TelegramHTML(activity.Activity{
		Event:   activity.EventPullRequestReview,
		Repo:    "owner/repo",
		Action:  "commented",
		Number:  12,
		Title:   "Add login",
		Actor:   "octocat",
		Summary: longText,
		URL:     "https://github.com/owner/repo/pull/12#pullrequestreview-1",
	})

	if strings.Contains(reviewMsg, "Comment truncated. Open on GitHub for full text.") {
		t.Fatalf("non-comment summary should not include comment truncation notice:\n%s", reviewMsg)
	}
	if !strings.Contains(reviewMsg, strings.Repeat("a", maxCommentRunes+20)) {
		t.Fatalf("review summary should not use comment-specific truncation:\n%s", reviewMsg)
	}
}
