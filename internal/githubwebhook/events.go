package githubwebhook

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/yorukot/gitgram/internal/activity"
)

var ErrIgnored = errors.New("github webhook event ignored")

func ParseEvent(eventType, deliveryID string, body []byte) (activity.Activity, error) {
	switch eventType {
	case activity.EventPush:
		return parsePush(deliveryID, body)
	case activity.EventPullRequest:
		return parsePullRequest(deliveryID, body)
	case activity.EventIssues:
		return parseIssues(deliveryID, body)
	case activity.EventIssueComment:
		return parseIssueComment(deliveryID, body)
	case activity.EventPullRequestReview:
		return parsePullRequestReview(deliveryID, body)
	case activity.EventRelease:
		return parseRelease(deliveryID, body)
	case activity.EventWorkflowRun:
		return parseWorkflowRun(deliveryID, body)
	case "ping":
		return activity.Activity{}, ErrIgnored
	default:
		return activity.Activity{}, ErrIgnored
	}
}

type account struct {
	Login string `json:"login"`
}

type repository struct {
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

type pushPayload struct {
	Ref        string     `json:"ref"`
	Compare    string     `json:"compare"`
	Repository repository `json:"repository"`
	Sender     account    `json:"sender"`
	Pusher     struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		URL     string `json:"url"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"commits"`
}

func parsePush(deliveryID string, body []byte) (activity.Activity, error) {
	var p pushPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse push payload: %w", err)
	}

	commits := make([]activity.Commit, 0, len(p.Commits))
	for _, commit := range p.Commits {
		commits = append(commits, activity.Commit{
			SHA:     commit.ID,
			Message: firstLine(commit.Message),
			URL:     commit.URL,
			Author:  commit.Author.Name,
		})
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventPush,
		Repo:       p.Repository.FullName,
		Action:     "pushed",
		Actor:      firstNonEmpty(p.Sender.Login, p.Pusher.Name),
		URL:        p.Compare,
		Branch:     branchFromRef(p.Ref),
		Commits:    commits,
	}, nil
}

type pullRequestPayload struct {
	Action      string     `json:"action"`
	Number      int        `json:"number"`
	Repository  repository `json:"repository"`
	Sender      account    `json:"sender"`
	PullRequest struct {
		HTMLURL string  `json:"html_url"`
		Title   string  `json:"title"`
		Number  int     `json:"number"`
		Merged  bool    `json:"merged"`
		User    account `json:"user"`
		Head    struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
}

func parsePullRequest(deliveryID string, body []byte) (activity.Activity, error) {
	var p pullRequestPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse pull_request payload: %w", err)
	}

	if !oneOf(p.Action, "opened", "closed", "reopened", "synchronize") {
		return activity.Activity{}, ErrIgnored
	}

	action := p.Action
	if p.Action == "closed" && p.PullRequest.Merged {
		action = "merged"
	}
	if p.Action == "synchronize" {
		action = "updated"
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventPullRequest,
		Repo:       p.Repository.FullName,
		Action:     action,
		Title:      p.PullRequest.Title,
		Actor:      firstNonEmpty(p.Sender.Login, p.PullRequest.User.Login),
		URL:        p.PullRequest.HTMLURL,
		Branch:     p.PullRequest.Head.Ref,
		BaseBranch: p.PullRequest.Base.Ref,
		Number:     firstNonZero(p.PullRequest.Number, p.Number),
	}, nil
}

type issuesPayload struct {
	Action     string     `json:"action"`
	Repository repository `json:"repository"`
	Sender     account    `json:"sender"`
	Issue      struct {
		HTMLURL     string           `json:"html_url"`
		Title       string           `json:"title"`
		Number      int              `json:"number"`
		User        account          `json:"user"`
		PullRequest *json.RawMessage `json:"pull_request,omitempty"`
	} `json:"issue"`
}

func parseIssues(deliveryID string, body []byte) (activity.Activity, error) {
	var p issuesPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse issues payload: %w", err)
	}
	if p.Issue.PullRequest != nil {
		return activity.Activity{}, ErrIgnored
	}
	if !oneOf(p.Action, "opened", "closed", "reopened") {
		return activity.Activity{}, ErrIgnored
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventIssues,
		Repo:       p.Repository.FullName,
		Action:     p.Action,
		Subject:    "issue",
		Title:      p.Issue.Title,
		Actor:      firstNonEmpty(p.Sender.Login, p.Issue.User.Login),
		URL:        p.Issue.HTMLURL,
		Number:     p.Issue.Number,
	}, nil
}

type issueCommentPayload struct {
	Action     string     `json:"action"`
	Repository repository `json:"repository"`
	Sender     account    `json:"sender"`
	Issue      struct {
		HTMLURL     string           `json:"html_url"`
		Title       string           `json:"title"`
		Number      int              `json:"number"`
		User        account          `json:"user"`
		PullRequest *json.RawMessage `json:"pull_request,omitempty"`
	} `json:"issue"`
	Comment struct {
		HTMLURL string  `json:"html_url"`
		Body    string  `json:"body"`
		User    account `json:"user"`
	} `json:"comment"`
}

func parseIssueComment(deliveryID string, body []byte) (activity.Activity, error) {
	var p issueCommentPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse issue_comment payload: %w", err)
	}
	if !oneOf(p.Action, "created", "edited") {
		return activity.Activity{}, ErrIgnored
	}

	subject := "issue"
	if p.Issue.PullRequest != nil {
		subject = "pull request"
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventIssueComment,
		Repo:       p.Repository.FullName,
		Action:     p.Action,
		Subject:    subject,
		Title:      p.Issue.Title,
		Actor:      firstNonEmpty(p.Sender.Login, p.Comment.User.Login, p.Issue.User.Login),
		URL:        firstNonEmpty(p.Comment.HTMLURL, p.Issue.HTMLURL),
		Number:     p.Issue.Number,
		Summary:    strings.TrimSpace(p.Comment.Body),
	}, nil
}

type pullRequestReviewPayload struct {
	Action      string     `json:"action"`
	Repository  repository `json:"repository"`
	Sender      account    `json:"sender"`
	PullRequest struct {
		HTMLURL string  `json:"html_url"`
		Title   string  `json:"title"`
		Number  int     `json:"number"`
		User    account `json:"user"`
	} `json:"pull_request"`
	Review struct {
		HTMLURL string  `json:"html_url"`
		State   string  `json:"state"`
		Body    string  `json:"body"`
		User    account `json:"user"`
	} `json:"review"`
}

func parsePullRequestReview(deliveryID string, body []byte) (activity.Activity, error) {
	var p pullRequestReviewPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse pull_request_review payload: %w", err)
	}
	if p.Action != "submitted" {
		return activity.Activity{}, ErrIgnored
	}

	action := strings.TrimSpace(strings.ToLower(p.Review.State))
	if action == "" {
		action = "reviewed"
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventPullRequestReview,
		Repo:       p.Repository.FullName,
		Action:     action,
		Title:      p.PullRequest.Title,
		Actor:      firstNonEmpty(p.Sender.Login, p.Review.User.Login, p.PullRequest.User.Login),
		URL:        firstNonEmpty(p.Review.HTMLURL, p.PullRequest.HTMLURL),
		Number:     p.PullRequest.Number,
		Summary:    strings.TrimSpace(p.Review.Body),
	}, nil
}

type releasePayload struct {
	Action     string     `json:"action"`
	Repository repository `json:"repository"`
	Sender     account    `json:"sender"`
	Release    struct {
		HTMLURL string  `json:"html_url"`
		Name    string  `json:"name"`
		TagName string  `json:"tag_name"`
		Author  account `json:"author"`
	} `json:"release"`
}

func parseRelease(deliveryID string, body []byte) (activity.Activity, error) {
	var p releasePayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse release payload: %w", err)
	}
	if p.Action != "published" {
		return activity.Activity{}, ErrIgnored
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventRelease,
		Repo:       p.Repository.FullName,
		Action:     "published",
		Title:      firstNonEmpty(p.Release.Name, p.Release.TagName),
		Actor:      firstNonEmpty(p.Sender.Login, p.Release.Author.Login),
		URL:        p.Release.HTMLURL,
	}, nil
}

type workflowRunPayload struct {
	Action      string     `json:"action"`
	Repository  repository `json:"repository"`
	Sender      account    `json:"sender"`
	WorkflowRun struct {
		HTMLURL    string  `json:"html_url"`
		Name       string  `json:"name"`
		HeadBranch string  `json:"head_branch"`
		Conclusion string  `json:"conclusion"`
		Actor      account `json:"actor"`
	} `json:"workflow_run"`
}

func parseWorkflowRun(deliveryID string, body []byte) (activity.Activity, error) {
	var p workflowRunPayload
	if err := decode(body, &p); err != nil {
		return activity.Activity{}, fmt.Errorf("parse workflow_run payload: %w", err)
	}

	conclusion := strings.ToLower(strings.TrimSpace(p.WorkflowRun.Conclusion))
	if p.Action != "completed" || conclusion != "failure" {
		return activity.Activity{}, ErrIgnored
	}

	return activity.Activity{
		DeliveryID: deliveryID,
		Event:      activity.EventWorkflowRun,
		Repo:       p.Repository.FullName,
		Action:     "failed",
		Title:      p.WorkflowRun.Name,
		Actor:      firstNonEmpty(p.Sender.Login, p.WorkflowRun.Actor.Login),
		URL:        p.WorkflowRun.HTMLURL,
		Branch:     p.WorkflowRun.HeadBranch,
		Conclusion: conclusion,
	}, nil
}

func decode(body []byte, v any) error {
	if len(body) == 0 {
		return errors.New("empty body")
	}
	if err := json.Unmarshal(body, v); err != nil {
		return err
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
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

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func branchFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	for _, prefix := range []string{"refs/heads/", "refs/tags/"} {
		if strings.HasPrefix(ref, prefix) {
			return strings.TrimPrefix(ref, prefix)
		}
	}
	return ref
}

func firstLine(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	line, _, _ := strings.Cut(value, "\n")
	return strings.TrimSpace(line)
}
