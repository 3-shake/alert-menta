package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func main() {
	// Get command line arguments
	var (
		repo        = flag.String("repo", "", "Repository name")
		owner       = flag.String("owner", "", "Repository owner")
		issueNumber = flag.Int("issue", 0, "Issue number")
		commentBody = flag.String("comment", "", "Comment body")
		token       = flag.String("token", "", "GitHub token")
	)
	flag.Parse()

	if *repo == "" || *owner == "" || *issueNumber == 0 || *token == "" || *commentBody == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	// Create GitHub client with OAuth2 token
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Retrieve specified GitHub Issues and their comments
	issue, _, _ := client.Issues.Get(ctx, *owner, *repo, *issueNumber)
	comments, res, err := client.Issues.ListComments(ctx, *owner, *repo, *issueNumber, &github.IssueListCommentsOptions{Direction: "asc"})

	logger.Println("Title:", issue.GetTitle())
	logger.Println("Body:", issue.GetBody())
	logger.Println("A number of comments:", issue.GetComments())
	logger.Println(len(comments), res, err)
	for _, v := range comments {
		logger.Printf("%s: %s", *v.User.Login, *v.Body)
	}

	// Post a comment on the Issue
	comment := &github.IssueComment{Body: github.String(*commentBody)}
	_, _, err = client.Issues.CreateComment(ctx, *owner, *repo, *issueNumber, comment)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}
	logger.Printf("Comment created successfully on Issue %d", *issueNumber)
}
