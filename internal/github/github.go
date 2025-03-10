package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

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
		gh.logger.Fatalf("Error creating comment: %v", err)
	}
	gh.logger.Printf("Comment created successfully on Issue %d", gh.issueNumber)
	return err
}

func (gh *GitHubIssue) GetDefaultBranch() (string, error) {
	repoInfo, _, err := gh.client.Repositories.Get(gh.ctx, gh.owner, gh.repo)
	return repoInfo.GetDefaultBranch(), err
}

// ListFiles lists all files in a specific branch of a GitHub repository.
func (gh *GitHubIssue) ListFiles(branch string) ([]string, error) {
	tree, response, err := gh.client.Git.GetTree(gh.ctx, gh.owner, gh.repo, branch, true)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusForbidden {
			if strings.Contains(response.Header.Get("Retry-After"), "s") {
				return nil, fmt.Errorf("rate limited by GitHub API, recommended to use personal access token: %w", err)
			}

		}
		return nil, fmt.Errorf("failed to get tree for branch %s: %w", branch, err)
	}

	var fileNames []string
	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" {
			fileNames = append(fileNames, entry.GetPath())
		}
	}

	return fileNames, nil
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

func GetAllIssues(owner, repo, token string) []*GitHubIssue {
	// Create GitHub client with OAuth2 token
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	opt := &github.IssueListByRepoOptions{
		State:       "all",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allIssues []*GitHubIssue
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, opt)
		if err != nil {
			log.Fatal(err)
		}
		for _, issue := range issues {
			allIssues = append(allIssues, NewIssue(owner, repo, *issue.Number, token))
		}
		// allIssues = append(allIssues, issues...)
		if resp.NextPage == 0 {
			break
		}
		opt.ListOptions.Page = resp.NextPage
	}
	return allIssues
}
