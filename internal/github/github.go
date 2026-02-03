package github

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type GitHubIssue struct {
	owner       string
	repo        string
	issueNumber int
	cache       *github.Issue
	token       string
	client      *github.Client
	ctx         context.Context
	logger      *log.Logger
}

func (gh *GitHubIssue) GetIssue() (*github.Issue, error) {
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

func (gh *GitHubIssue) GetBody() (*string, error) {
	issue, err := gh.GetIssue()
	if err != nil {
		return nil, err
	}
	return issue.Body, err
}

func (gh *GitHubIssue) GetTitle() (*string, error) {
	issue, err := gh.GetIssue()
	if err != nil {
		return nil, err
	}
	return issue.Title, err
}

func (gh *GitHubIssue) GetLabels() ([]string, error) {
	issue, err := gh.GetIssue()
	if err != nil {
		return nil, err
	}
	var labels []string
	for _, label := range issue.Labels {
		if label.Name != nil {
			labels = append(labels, *label.Name)
		}
	}
	return labels, nil
}

func (gh *GitHubIssue) GetComments() ([]*github.IssueComment, error) {
	// Options
	opt := &github.IssueListCommentsOptions{Direction: "asc", Sort: "created"}
	opt.Page = 1
	opt.PerPage = 100

	comments, _, err := gh.client.Issues.ListComments(gh.ctx, gh.owner, gh.repo, gh.issueNumber, opt)
	return comments, err
}

func (gh *GitHubIssue) PostComment(commentBody string) error {
	comment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := gh.client.Issues.CreateComment(gh.ctx, gh.owner, gh.repo, gh.issueNumber, comment)
	if err != nil {
		return fmt.Errorf("error creating comment: %w", err)
	}
	gh.logger.Printf("Comment created successfully on Issue %d", gh.issueNumber)
	return nil
}

func (gh *GitHubIssue) AddLabels(labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	_, _, err := gh.client.Issues.AddLabelsToIssue(gh.ctx, gh.owner, gh.repo, gh.issueNumber, labels)
	if err != nil {
		return fmt.Errorf("error adding labels: %w", err)
	}
	gh.logger.Printf("Labels %v added successfully to Issue %d", labels, gh.issueNumber)
	return nil
}

func NewIssue(owner string, repo string, issueNumber int, token string) *GitHubIssue {
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

	// Create a new GitHubIssue instance
	issue := &GitHubIssue{owner: owner, repo: repo, issueNumber: issueNumber, token: token, client: client, ctx: ctx, logger: logger}
	return issue
}
