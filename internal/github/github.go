package github

import (
	"context"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GitHubIssues struct {
	owner       string
	repo        string
	issueNumber int
	cache       *github.Issue
	token       string
	client      *github.Client
	ctx         context.Context
	logger      *log.Logger
}

func (gh *GitHubIssues) GetIssue() (*github.Issue, error) {
	// Only the first call retrieves information from GitHub, all other calls use cache
	if gh.cache == nil {
		issue, _, err := gh.client.Issues.Get(gh.ctx, gh.owner, gh.repo, gh.issueNumber)
		if err != nil {
			return nil, err
		}
		gh.cache = issue
	}
	return gh.cache, nil
}

func (gh *GitHubIssues) GetBody() (*string, error) {
	issue, err := gh.GetIssue()
	if err != nil {
		return nil, err
	}
	return issue.Body, err
}

func (gh *GitHubIssues) GetTitle() (*string, error) {
	issue, err := gh.GetIssue()
	if err != nil {
		return nil, err
	}
	return issue.Title, err
}

func (gh *GitHubIssues) GetComments() ([]*github.IssueComment, error) {
	// Options
	opt := &github.IssueListCommentsOptions{Direction: "asc", Sort: "created"}
	opt.Page = 1
	opt.PerPage = 100

	comments, _, err := gh.client.Issues.ListComments(gh.ctx, gh.owner, gh.repo, gh.issueNumber, opt)
	return comments, err
}

func (gh *GitHubIssues) PostComment(commentBody string) error {
	comment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := gh.client.Issues.CreateComment(gh.ctx, gh.owner, gh.repo, gh.issueNumber, comment)
	if err != nil {
		gh.logger.Fatalf("Error creating comment: %s", err)
	}
	gh.logger.Printf("Comment created successfully on Issue %d", gh.issueNumber)
	return err
}

func NewIssue(owner string, repo string, issueNumber int, token string) *GitHubIssues {
	// Create GitHub client with OAuth2 token
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta github] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Create a new GitHubIssues instance
	issue := &GitHubIssues{owner: owner, repo: repo, issueNumber: issueNumber, token: token, client: client, ctx: ctx, logger: logger}
	return issue
}
