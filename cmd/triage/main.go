package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/triage"
	"github.com/3-shake/alert-menta/internal/utils"
)

func main() {
	// Parse command line flags
	repo := flag.String("repo", "", "Repository name")
	owner := flag.String("owner", "", "Repository owner")
	issueNumber := flag.Int("issue", 0, "Issue number")
	configFile := flag.String("config", ".alert-menta.user.yaml", "Configuration file")
	ghToken := flag.String("github-token", "", "GitHub token (or set GITHUB_TOKEN env)")
	apiKey := flag.String("api-key", "", "AI API key (or set OPENAI_API_KEY env)")
	dryRun := flag.Bool("dry-run", false, "Print triage result without applying labels")
	flag.Parse()

	// Get tokens from environment if not provided
	githubToken := *ghToken
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}

	aiAPIKey := *apiKey
	if aiAPIKey == "" {
		aiAPIKey = os.Getenv("OPENAI_API_KEY")
		if aiAPIKey == "" {
			aiAPIKey = os.Getenv("ANTHROPIC_API_KEY")
		}
	}

	// Validate required flags
	if *repo == "" || *owner == "" || *issueNumber == 0 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if githubToken == "" {
		log.Fatal("GitHub token is required (--github-token or GITHUB_TOKEN env)")
	}

	if aiAPIKey == "" {
		log.Fatal("AI API key is required (--api-key or OPENAI_API_KEY/ANTHROPIC_API_KEY env)")
	}

	logger := log.New(
		os.Stdout, "[alert-menta triage] ",
		log.Ldate|log.Ltime|log.Lmsgprefix,
	)

	// Load configuration
	cfg, err := utils.NewConfig(*configFile)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	// Check if triage is enabled
	if !cfg.Triage.Enabled {
		logger.Println("Triage is disabled in config")
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

	existingLabels, err := issue.GetLabels()
	if err != nil {
		logger.Fatalf("Failed to get issue labels: %v", err)
	}

	// Create AI client
	aiClient, err := getAIClient(aiAPIKey, cfg)
	if err != nil {
		logger.Fatalf("Failed to create AI client: %v", err)
	}

	// Create triage config from utils config
	triageConfig := &triage.Config{
		Enabled:             cfg.Triage.Enabled,
		AutoLabel:           cfg.Triage.AutoLabel,
		AutoComment:         cfg.Triage.AutoComment,
		ConfidenceThreshold: cfg.Triage.ConfidenceThreshold,
	}

	// Copy label definitions
	for _, label := range cfg.Triage.Labels.Priority {
		triageConfig.Labels.Priority = append(triageConfig.Labels.Priority, triage.LabelDefinition{
			Name:     label.Name,
			Criteria: label.Criteria,
		})
	}
	for _, label := range cfg.Triage.Labels.Category {
		triageConfig.Labels.Category = append(triageConfig.Labels.Category, triage.LabelDefinition{
			Name:     label.Name,
			Criteria: label.Criteria,
		})
	}

	// Use default config if no labels defined
	if len(triageConfig.Labels.Priority) == 0 && len(triageConfig.Labels.Category) == 0 {
		defaultConfig := triage.DefaultConfig()
		triageConfig.Labels = defaultConfig.Labels
	}

	if triageConfig.ConfidenceThreshold == 0 {
		triageConfig.ConfidenceThreshold = 0.7
	}

	// Create triager and run triage
	triager := triage.NewTriager(triageConfig, aiClient)

	logger.Printf("Triaging Issue #%d: %s", *issueNumber, *title)

	result, err := triager.Triage(*title, *body, existingLabels)
	if err != nil {
		logger.Fatalf("Triage failed: %v", err)
	}

	logger.Printf("Triage result: Priority=%s (%.0f%%), Category=%s (%.0f%%)",
		result.Priority.Label, result.Priority.Confidence*100,
		result.Category.Label, result.Category.Confidence*100)
	logger.Printf("Reasoning: %s", result.Reasoning)

	if *dryRun {
		fmt.Println("\n=== Triage Result (dry-run) ===")
		fmt.Println(triager.FormatComment(result))
		fmt.Printf("\nLabels to apply: %v\n", triager.GetLabelsToApply(result))
		os.Exit(0)
	}

	// Apply labels if enabled
	if triageConfig.AutoLabel {
		labelsToApply := triager.GetLabelsToApply(result)
		if len(labelsToApply) > 0 {
			if err := issue.AddLabels(labelsToApply); err != nil {
				logger.Printf("Warning: failed to add labels: %v", err)
			} else {
				logger.Printf("Applied labels: %v", labelsToApply)
			}
		}
	}

	// Post comment if enabled
	if triageConfig.AutoComment {
		comment := triager.FormatComment(result)
		if err := issue.PostComment(comment); err != nil {
			logger.Fatalf("Failed to post comment: %v", err)
		}
		logger.Printf("Posted triage comment to Issue #%d", *issueNumber)
	}
}

func getAIClient(apiKey string, cfg *utils.Config) (ai.Ai, error) {
	switch cfg.Ai.Provider {
	case "openai":
		return ai.NewOpenAIClient(apiKey, cfg.Ai.OpenAI.Model), nil
	case "anthropic":
		return ai.NewAnthropicClient(apiKey, cfg.Ai.Anthropic.Model), nil
	case "vertexai":
		return ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
	default:
		return nil, fmt.Errorf("invalid provider: %s", cfg.Ai.Provider)
	}
}
