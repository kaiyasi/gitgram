package formatter

import (
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/yorukot/gitgram/internal/activity"
)

const (
	maxCommitLines      = 5
	maxCommentRunes     = 300
	maxTelegramRunes    = 3900
	truncatedSuffixText = "\n\n(truncated)"
)

func TelegramHTML(a activity.Activity) string {
	var b strings.Builder

	switch a.Event {
	case activity.EventPush:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> push to "+code(a.Branch))
		writeActor(&b, a.Actor)
		writeBlank(&b)
		writeCommits(&b, a.Commits)
		writeLink(&b, a.URL)
	case activity.EventPullRequest:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> pull request "+esc(a.Action))
		writeNumberedTitle(&b, a.Number, a.Title)
		writeActor(&b, a.Actor)
		writeBranchPair(&b, a.Branch, a.BaseBranch)
		writeLink(&b, a.URL)
	case activity.EventIssues:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> issue "+esc(a.Action))
		writeNumberedTitle(&b, a.Number, a.Title)
		writeActor(&b, a.Actor)
		writeLink(&b, a.URL)
	case activity.EventIssueComment:
		subject := firstNonEmpty(a.Subject, "issue")
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> "+esc(subject)+" comment "+esc(a.Action))
		writeNumberedTitle(&b, a.Number, a.Title)
		writeActor(&b, a.Actor)
		if subject == "pull request" {
			writeCommentSummary(&b, a.Summary)
		} else {
			writeSummary(&b, a.Summary)
		}
		writeLink(&b, a.URL)
	case activity.EventPullRequestReview:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> pull request review "+esc(humanize(a.Action)))
		writeNumberedTitle(&b, a.Number, a.Title)
		writeActor(&b, a.Actor)
		writeSummary(&b, a.Summary)
		writeLink(&b, a.URL)
	case activity.EventRelease:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> release "+esc(a.Action))
		writeLine(&b, esc(a.Title))
		writeActor(&b, a.Actor)
		writeLink(&b, a.URL)
	case activity.EventWorkflowRun:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> workflow failed")
		if a.Branch != "" {
			writeLine(&b, esc(a.Title)+" on "+code(a.Branch))
		} else {
			writeLine(&b, esc(a.Title))
		}
		writeActor(&b, a.Actor)
		writeLink(&b, a.URL)
	default:
		writeLine(&b, "<b>"+esc(a.Repo)+"</b> "+esc(a.Event)+" "+esc(a.Action))
		writeLine(&b, esc(a.Title))
		writeActor(&b, a.Actor)
		writeSummary(&b, a.Summary)
		writeLink(&b, a.URL)
	}

	return trimTelegramMessage(strings.TrimSpace(b.String()))
}

func writeCommits(b *strings.Builder, commits []activity.Commit) {
	if len(commits) == 0 {
		writeLine(b, "No commits included in payload.")
		writeBlank(b)
		return
	}

	limit := len(commits)
	if limit > maxCommitLines {
		limit = maxCommitLines
	}

	for i := 0; i < limit; i++ {
		commit := commits[i]
		writeLine(b, code(shortSHA(commit.SHA))+" "+esc(commit.Message))
	}
	if len(commits) > limit {
		writeLine(b, fmt.Sprintf("... and %d more commits", len(commits)-limit))
	}
	writeBlank(b)
}

func writeNumberedTitle(b *strings.Builder, number int, title string) {
	if number > 0 {
		writeLine(b, fmt.Sprintf("#%d %s", number, esc(title)))
		return
	}
	writeLine(b, esc(title))
}

func writeActor(b *strings.Builder, actor string) {
	if actor != "" {
		writeLine(b, "by "+code(actor))
	}
}

func writeBranchPair(b *strings.Builder, head, base string) {
	if head != "" && base != "" {
		writeLine(b, code(head)+" -> "+code(base))
	}
}

func writeSummary(b *strings.Builder, summary string) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		writeBlank(b)
		return
	}
	writeBlank(b)
	writeLine(b, esc(summary))
	writeBlank(b)
}

func writeCommentSummary(b *strings.Builder, summary string) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		writeBlank(b)
		return
	}

	truncated := utf8.RuneCountInString(summary) > maxCommentRunes
	if truncated {
		summary = truncateRunes(summary, maxCommentRunes)
	}

	writeBlank(b)
	writeLine(b, esc(summary))
	if truncated {
		writeLine(b, "<i>Comment truncated. Open on GitHub for full text.</i>")
	}
	writeBlank(b)
}

func writeLink(b *strings.Builder, url string) {
	if url == "" {
		return
	}
	writeLine(b, `<a href="`+esc(url)+`">Open on GitHub</a>`)
}

func writeLine(b *strings.Builder, value string) {
	if value == "" {
		return
	}
	b.WriteString(value)
	b.WriteByte('\n')
}

func writeBlank(b *strings.Builder) {
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
}

func esc(value string) string {
	return html.EscapeString(strings.TrimSpace(value))
}

func code(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "<code>unknown</code>"
	}
	return "<code>" + esc(value) + "</code>"
}

func humanize(value string) string {
	return strings.ReplaceAll(value, "_", " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func shortSHA(sha string) string {
	sha = strings.TrimSpace(sha)
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || utf8.RuneCountInString(value) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[:max]) + "..."
}

func trimTelegramMessage(value string) string {
	if utf8.RuneCountInString(value) <= maxTelegramRunes {
		return value
	}
	runes := []rune(value)
	limit := maxTelegramRunes - utf8.RuneCountInString(truncatedSuffixText)
	if limit < 0 {
		limit = maxTelegramRunes
	}
	return strings.TrimSpace(string(runes[:limit])) + truncatedSuffixText
}
