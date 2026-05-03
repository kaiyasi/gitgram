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
		"do not parse me",
		`href="https://github.com/owner/repo/issues/42?x=1&amp;y=2"`,
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message does not contain %q:\n%s", want, msg)
		}
	}
}

func TestTelegramHTMLSummarizesPullRequestCommentAsPlainText(t *testing.T) {
	msg := TelegramHTML(activity.Activity{
		Event:   activity.EventIssueComment,
		Repo:    "owner/repo",
		Action:  "created",
		Subject: "pull request",
		Number:  12,
		Title:   "Add login",
		Actor:   "octocat",
		Summary: strings.Join([]string{
			"<!-- This is an auto-generated comment: summarize by coderabbit.ai -->",
			"",
			"Deploying with &nbsp;<a href=\"https://workers.dev\"><img alt=\"Cloudflare Workers\" src=\"logo.svg\"></a> &nbsp;Cloudflare Workers",
			"",
			"Please check [the preview](https://example.com/preview?a=1&b=2).",
			"",
			"Learn more about integrating Git with Workers.",
		}, "\n"),
		URL: "https://github.com/owner/repo/pull/12#issuecomment-1",
	})

	for _, want := range []string{
		"Deploying with Cloudflare Workers",
		"Please check the preview.",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message does not contain %q:\n%s", want, msg)
		}
	}

	for _, unwanted := range []string{
		"auto-generated comment",
		"<img",
		"https://example.com/preview",
		"Learn more",
	} {
		if strings.Contains(msg, unwanted) {
			t.Fatalf("message should not contain %q:\n%s", unwanted, msg)
		}
	}
}

func TestTelegramHTMLDoesNotRenderCommentMarkdown(t *testing.T) {
	msg := TelegramHTML(activity.Activity{
		Event:   activity.EventIssueComment,
		Repo:    "owner/repo",
		Action:  "created",
		Subject: "issue",
		Number:  42,
		Title:   "Cannot login",
		Actor:   "octocat",
		Summary: `**bold** [docs](https://example.com)`,
		URL:     "https://github.com/owner/repo/issues/42#issuecomment-1",
	})

	if strings.Contains(msg, "<b>bold</b>") || strings.Contains(msg, `<a href="https://example.com">docs</a>`) || strings.Contains(msg, "https://example.com") {
		t.Fatalf("comment markdown should not be rendered:\n%s", msg)
	}
	if !strings.Contains(msg, `bold docs`) {
		t.Fatalf("comment markdown should be summarized as plain text:\n%s", msg)
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

func TestTelegramHTMLTruncatesSummaries(t *testing.T) {
	longText := strings.Repeat("a", maxSummaryRunes+20)

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

	if strings.Contains(commentMsg, strings.Repeat("a", maxSummaryRunes+1)) {
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

	if strings.Contains(issueCommentMsg, strings.Repeat("a", maxSummaryRunes+1)) {
		t.Fatalf("issue comment message should be truncated:\n%s", issueCommentMsg)
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

	if strings.Contains(reviewMsg, strings.Repeat("a", maxSummaryRunes+1)) {
		t.Fatalf("review message should be truncated:\n%s", reviewMsg)
	}
}
