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

// cliConfig holds command-line configuration
type cliConfig struct {
	repo        string
	owner       string
	issueNumber int
	configFile  string
	githubToken string
	aiAPIKey    string
	dryRun      bool
}

func main() {
	cfg := parseFlags()
	logger := log.New(os.Stdout, "[alert-menta triage] ", log.Ldate|log.Ltime|log.Lmsgprefix)

	if err := run(cfg, logger); err != nil {
		logger.Fatalf("Error: %v", err)
	}
}

func parseFlags() *cliConfig {
	cfg := &cliConfig{}
	flag.StringVar(&cfg.repo, "repo", "", "Repository name")
	flag.StringVar(&cfg.owner, "owner", "", "Repository owner")
	flag.IntVar(&cfg.issueNumber, "issue", 0, "Issue number")
	flag.StringVar(&cfg.configFile, "config", ".alert-menta.user.yaml", "Configuration file")
	ghToken := flag.String("github-token", "", "GitHub token (or set GITHUB_TOKEN env)")
	apiKey := flag.String("api-key", "", "AI API key (or set OPENAI_API_KEY env)")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "Print triage result without applying labels")
	flag.Parse()

	cfg.githubToken = getEnvOrDefault(*ghToken, "GITHUB_TOKEN")
	cfg.aiAPIKey = getEnvOrDefault(*apiKey, "OPENAI_API_KEY", "ANTHROPIC_API_KEY")

	return cfg
}

func getEnvOrDefault(value string, envKeys ...string) string {
	if value != "" {
		return value
	}
	for _, key := range envKeys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return ""
}

func run(cfg *cliConfig, logger *log.Logger) error {
	if err := validateConfig(cfg); err != nil {
		flag.PrintDefaults()
		return err
	}

	appConfig, err := utils.NewConfig(cfg.configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !appConfig.Triage.Enabled {
		logger.Println("Triage is disabled in config")
		return nil
	}

	issue := github.NewIssue(cfg.owner, cfg.repo, cfg.issueNumber, cfg.githubToken)
	title, body, existingLabels, err := getIssueInfo(issue)
	if err != nil {
		return err
	}

	aiClient, err := getAIClient(cfg.aiAPIKey, appConfig)
	if err != nil {
		return fmt.Errorf("creating AI client: %w", err)
	}

	triageConfig := buildTriageConfig(appConfig)
	triager := triage.NewTriager(triageConfig, aiClient)

	logger.Printf("Triaging Issue #%d: %s", cfg.issueNumber, title)

	result, err := triager.Triage(title, body, existingLabels)
	if err != nil {
		return fmt.Errorf("triage failed: %w", err)
	}

	logTriageResult(logger, result)

	if cfg.dryRun {
		printDryRunResult(triager, result)
		return nil
	}

	return applyTriageResult(triageConfig, triager, result, issue, cfg.issueNumber, logger)
}

func validateConfig(cfg *cliConfig) error {
	if cfg.repo == "" || cfg.owner == "" || cfg.issueNumber == 0 {
		return fmt.Errorf("repo, owner, and issue number are required")
	}
	if cfg.githubToken == "" {
		return fmt.Errorf("github token is required (--github-token or GITHUB_TOKEN env)")
	}
	if cfg.aiAPIKey == "" {
		return fmt.Errorf("ai API key is required (--api-key or OPENAI_API_KEY/ANTHROPIC_API_KEY env)")
	}
	return nil
}

func getIssueInfo(issue *github.GitHubIssue) (string, string, []string, error) {
	title, err := issue.GetTitle()
	if err != nil {
		return "", "", nil, fmt.Errorf("getting issue title: %w", err)
	}
	body, err := issue.GetBody()
	if err != nil {
		return "", "", nil, fmt.Errorf("getting issue body: %w", err)
	}
	labels, err := issue.GetLabels()
	if err != nil {
		return "", "", nil, fmt.Errorf("getting issue labels: %w", err)
	}
	return *title, *body, labels, nil
}

func buildTriageConfig(cfg *utils.Config) *triage.Config {
	triageConfig := &triage.Config{
		Enabled:             cfg.Triage.Enabled,
		AutoLabel:           cfg.Triage.AutoLabel,
		AutoComment:         cfg.Triage.AutoComment,
		ConfidenceThreshold: cfg.Triage.ConfidenceThreshold,
	}

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

	if len(triageConfig.Labels.Priority) == 0 && len(triageConfig.Labels.Category) == 0 {
		defaultConfig := triage.DefaultConfig()
		triageConfig.Labels = defaultConfig.Labels
	}

	if triageConfig.ConfidenceThreshold == 0 {
		triageConfig.ConfidenceThreshold = 0.7
	}

	return triageConfig
}

func logTriageResult(logger *log.Logger, result *triage.Result) {
	logger.Printf("Triage result: Priority=%s (%.0f%%), Category=%s (%.0f%%)",
		result.Priority.Label, result.Priority.Confidence*100,
		result.Category.Label, result.Category.Confidence*100)
	logger.Printf("Reasoning: %s", result.Reasoning)
}

func printDryRunResult(triager *triage.Triager, result *triage.Result) {
	fmt.Println("\n=== Triage Result (dry-run) ===")
	fmt.Println(triager.FormatComment(result))
	fmt.Printf("\nLabels to apply: %v\n", triager.GetLabelsToApply(result))
}

func applyTriageResult(cfg *triage.Config, triager *triage.Triager, result *triage.Result, issue *github.GitHubIssue, issueNumber int, logger *log.Logger) error {
	if cfg.AutoLabel {
		if err := applyLabels(triager, result, issue, logger); err != nil {
			logger.Printf("Warning: failed to add labels: %v", err)
		}
	}

	if cfg.AutoComment {
		comment := triager.FormatComment(result)
		if err := issue.PostComment(comment); err != nil {
			return fmt.Errorf("posting comment: %w", err)
		}
		logger.Printf("Posted triage comment to Issue #%d", issueNumber)
	}

	return nil
}

func applyLabels(triager *triage.Triager, result *triage.Result, issue *github.GitHubIssue, logger *log.Logger) error {
	labelsToApply := triager.GetLabelsToApply(result)
	if len(labelsToApply) == 0 {
		return nil
	}
	if err := issue.AddLabels(labelsToApply); err != nil {
		return err
	}
	logger.Printf("Applied labels: %v", labelsToApply)
	return nil
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
