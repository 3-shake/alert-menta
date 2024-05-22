package main

import (
	"flag"
	"log"
	"os"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
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

	// Create GitHub Issues instance
	issue := github.NewIssue(*owner, *repo, *issueNumber, *token)

	// Get Issue's info
	title, _ := issue.GetTitle()
	body, _ := issue.GetBody()
	logger.Println("Title:", *title)
	logger.Println("Body:", *body)
	user_prompt := "Title:" + *title + "\n"
	user_prompt += "Body:" + *body + "\n"

	comments, _ := issue.GetComments()
	for _, v := range comments {
		logger.Printf("%s: %s", *v.User.Login, *v.Body)
		user_prompt += *v.User.Login + ":" + *v.Body + "\n"
	}

	// Set system prompt
	prompt := "The following is the GitHub Issue and comments on it. Please summarize the conversation and suggest what issues need to be resolved.\n"

	// Get response from OpenAI
	logger.Println("Prompt:", prompt+user_prompt)
	ai := ai.NewOpenAIClient("", "gpt-3.5-turbo")
	comment, _ := ai.GetResponse(prompt + user_prompt)

	// Post a comment on the Issue
	err := issue.PostComment(comment)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}
	logger.Printf("Comment created successfully on Issue %d", *issueNumber)
}
