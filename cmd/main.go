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
		configFile  = flag.String("config", "./internal/config/config.yaml", "Configuration file")
		gh_token    = flag.String("github-token", "", "GitHub token")
		oai_key     = flag.String("api-key", "", "OpenAI api key")
	)
	flag.Parse()

	if *repo == "" || *owner == "" || *issueNumber == 0 || *gh_token == "" || *oai_key == "" || *command == "" {
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
	issue := github.NewIssue(*owner, *repo, *issueNumber, *gh_token)

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
	logger.Println("Prompt:", system_prompt, user_prompt)

	// Get response from OpenAI
	var aic ai.Ai
	if cfg.Ai.Provider == "openai" {
		aic = ai.NewOpenAIClient(*oai_key, cfg.Ai.OpenAI.Model)
		logger.Println("Using OpenAI API")
		logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
	} else {
		aic = ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
		logger.Println("Using VertexAI API")
		logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
	}
	comment, _ := aic.GetResponse(system_prompt + user_prompt)
	logger.Println("Response:", comment)

	// Post a comment on the Issue
	err = issue.PostComment(comment)
	if err != nil {
		logger.Fatalf("Error creating comment: %s", err)
	}
	logger.Printf("Comment created successfully on Issue %d", *issueNumber)
}
