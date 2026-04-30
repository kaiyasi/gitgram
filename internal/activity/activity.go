package activity

const (
	EventIssueComment      = "issue_comment"
	EventIssues            = "issues"
	EventPullRequest       = "pull_request"
	EventPullRequestReview = "pull_request_review"
	EventPush              = "push"
	EventRelease           = "release"
	EventWorkflowRun       = "workflow_run"
)

type Commit struct {
	SHA     string
	Message string
	URL     string
	Author  string
}

type Activity struct {
	DeliveryID string
	Event      string
	Repo       string
	Action     string
	Subject    string
	Title      string
	Actor      string
	URL        string
	Branch     string
	BaseBranch string
	Number     int
	Conclusion string
	Summary    string
	Commits    []Commit
}
