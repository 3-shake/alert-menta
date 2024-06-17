package main

import (
	"flag"
	"log"
	"os"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/utils"
)

func main() {
	// Get command line arguments
	var (
		repo        = flag.String("repo", "", "Repository name")
		owner       = flag.String("owner", "", "Repository owner")
		issueNumber = flag.Int("issue", 0, "Issue number")
		command     = flag.String("command", "", "Command to be executed by AI")
		token       = flag.String("token", "", "GitHub token")
		configFile  = flag.String("config", "./internal/config/config.yaml", "Configuration file")
	)
	flag.Parse()

	if *repo == "" || *owner == "" || *issueNumber == 0 || *token == "" || *command == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Initialize a logger
	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	var err error

	cfg, err := utils.NewConfig(*configFile)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}

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
		if *v.User.Login == "github-actions[bot]" {
			continue
		}
		logger.Printf("%s: %s", *v.User.Login, *v.Body)
		user_prompt += *v.User.Login + ":" + *v.Body + "\n"
	}

	// Set system prompt
	system_prompt := cfg.Ai.Commands[*command].System_prompt

	// Get response from OpenAI
	logger.Println("Prompt:", system_prompt, user_prompt)
	ai := ai.NewOpenAIClient("", cfg.Ai.Model)
	comment, _ := ai.GetResponse(system_prompt + user_prompt)
	logger.Println("Response:", comment)

	// Post a comment on the Issue
	err = issue.PostComment(comment)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}
	logger.Printf("Comment created successfully on Issue %d", *issueNumber)
}
