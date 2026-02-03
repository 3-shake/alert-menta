package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/3-shake/alert-menta/internal/firstresponse"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/utils"
)

func main() {
	// Parse command line flags
	repo := flag.String("repo", "", "Repository name")
	owner := flag.String("owner", "", "Repository owner")
	issueNumber := flag.Int("issue", 0, "Issue number")
	configFile := flag.String("config", ".alert-menta.user.yaml", "Configuration file")
	ghToken := flag.String("github-token", "", "GitHub token (or set GITHUB_TOKEN env)")
	dryRun := flag.Bool("dry-run", false, "Print guide without posting to GitHub")
	flag.Parse()

	// Get GitHub token from environment if not provided
	githubToken := *ghToken
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	// Validate required flags
	if *repo == "" || *owner == "" || *issueNumber == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if githubToken == "" {
		log.Fatal("GitHub token is required (--github-token or GITHUB_TOKEN env)")
	}

	logger := log.New(
		os.Stdout, "[alert-menta first-response] ",
		log.Ldate|log.Ltime|log.Lmsgprefix,
	)

	// Load configuration
	cfg, err := utils.NewConfig(*configFile)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	// Check if first response is enabled
	if !cfg.FirstResponse.Enabled {
		logger.Println("First response guide is disabled in config")
		os.Exit(0)
	}

	// Get issue information
	issue := github.NewIssue(*owner, *repo, *issueNumber, githubToken)

	title, err := issue.GetTitle()
	if err != nil {
		logger.Fatalf("Failed to get issue title: %v", err)
	}

	body, err := issue.GetBody()
	if err != nil {
		logger.Fatalf("Failed to get issue body: %v", err)
	}

	labels, err := issue.GetLabels()
	if err != nil {
		logger.Fatalf("Failed to get issue labels: %v", err)
	}

	// Check if should trigger based on labels
	if !firstresponse.ShouldTrigger(labels, cfg.FirstResponse.TriggerLabels) {
		logger.Printf("Issue labels %v don't match trigger labels %v, skipping",
			labels, cfg.FirstResponse.TriggerLabels)
		os.Exit(0)
	}

	// Check if guide already exists
	comments, err := issue.GetComments()
	if err != nil {
		logger.Fatalf("Failed to get comments: %v", err)
	}

	var commentBodies []string
	for _, c := range comments {
		if c.Body != nil {
			commentBodies = append(commentBodies, *c.Body)
		}
	}

	if firstresponse.HasExistingGuide(commentBodies) {
		logger.Println("First response guide already exists, skipping")
		os.Exit(0)
	}

	// Build command list from config
	var commands []firstresponse.CommandInfo
	for name, cmd := range cfg.Ai.Commands {
		commands = append(commands, firstresponse.CommandInfo{
			Name:        name,
			Description: cmd.Description,
		})
	}

	// Create guide generator
	frConfig := &firstresponse.Config{
		Enabled:       cfg.FirstResponse.Enabled,
		TriggerLabels: cfg.FirstResponse.TriggerLabels,
		DefaultGuide:  cfg.FirstResponse.DefaultGuide,
	}

	for _, g := range cfg.FirstResponse.Guides {
		frConfig.Guides = append(frConfig.Guides, firstresponse.GuideConfig{
			Severity:   g.Severity,
			Template:   g.Template,
			AutoNotify: g.AutoNotify,
		})
	}

	generator := firstresponse.NewGenerator(frConfig, commands)

	// Generate guide
	issueSummary := firstresponse.IssueSummary{
		Number: *issueNumber,
		Title:  *title,
		URL:    fmt.Sprintf("https://github.com/%s/%s/issues/%d", *owner, *repo, *issueNumber),
		Labels: labels,
	}

	guide, err := generator.Generate(issueSummary, *body)
	if err != nil {
		logger.Fatalf("Failed to generate guide: %v", err)
	}

	if *dryRun {
		fmt.Println("=== Generated Guide (dry-run) ===")
		fmt.Println(guide)
		os.Exit(0)
	}

	// Post guide as comment
	if err := issue.PostComment(guide); err != nil {
		logger.Fatalf("Failed to post guide comment: %v", err)
	}

	logger.Printf("First response guide posted to %s/%s#%d", *owner, *repo, *issueNumber)
}
