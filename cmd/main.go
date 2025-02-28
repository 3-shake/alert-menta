package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/3-shake/alert-menta/internal/ai"
	"github.com/3-shake/alert-menta/internal/github"
	"github.com/3-shake/alert-menta/internal/rag"
	"github.com/3-shake/alert-menta/internal/utils"
)

// Struct to hold the command-line arguments
type Config struct {
	repo        string
	owner       string
	issueNumber int
	intent      string
	command     string
	configFile  string
	ghToken     string
	oaiKey      string
	useRag      bool
}

type Neo4jConfig struct {
	uri           string
	username      string
	password      string
	fulltextIndex string
	vectorIndex   string
}

func main() {
	cfg := &Config{}
	neo4jcfg := &Neo4jConfig{}
	flag.StringVar(&cfg.repo, "repo", "", "Repository name")
	flag.StringVar(&cfg.owner, "owner", "", "Repository owner")
	flag.IntVar(&cfg.issueNumber, "issue", 0, "Issue number")
	flag.StringVar(&cfg.intent, "intent", "", "Question or intent for the 'ask' command")
	flag.StringVar(&cfg.command, "command", "", "Commands to be executed by AI. Commands defined in the configuration file are available.")
	flag.StringVar(&cfg.configFile, "config", "", "Configuration file")
	flag.StringVar(&cfg.ghToken, "github-token", "", "GitHub token")
	flag.StringVar(&cfg.oaiKey, "api-key", "", "OpenAI api key")
	flag.BoolVar(&cfg.useRag, "use-rag", false, "Use RAG model for response generation")
	flag.StringVar(&neo4jcfg.uri, "neo4j-uri", "", "Neo4j URI")
	flag.StringVar(&neo4jcfg.username, "neo4j-username", "", "Neo4j username")
	flag.StringVar(&neo4jcfg.password, "neo4j-password", "", "Neo4j password")
	flag.StringVar(&neo4jcfg.fulltextIndex, "fulltext-index", "keyword", "Neo4j fulltext index(default: keyword)")
	flag.StringVar(&neo4jcfg.vectorIndex, "vector-index", "vector", "Neo4j vector index(default: vector)")
	flag.Parse()

	if cfg.repo == "" || cfg.owner == "" || cfg.issueNumber == 0 || cfg.ghToken == "" || cfg.command == "" || cfg.configFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	var retriever *rag.Neo4jRetriever
	if cfg.useRag {
		flag.Parse()
		if neo4jcfg.uri == "" || neo4jcfg.username == "" || neo4jcfg.password == "" {
			fmt.Println("if -useRag is set, neo4j-uri, neo4j-username, and neo4j-password are required")
			fmt.Println("Usage: alert-menta -use-rag -neo4j-uri <uri> -neo4j-username <username> -neo4j-password <password>")
			fmt.Println("[-fulltext-index <fulltext-index>] [-vector-index <vector-index>]")
			os.Exit(1)
		}
		r, err := getNeo4jRetriever(neo4jcfg, log.New(os.Stdout, "[alert-menta main] ", log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix))
		retriever = r
		retriever.TestConnection()
		if err != nil {
			log.Fatalf("Error getting Neo4j retriever: %v", err)
		}
	}
	fmt.Println("Neo4j Retriever:", retriever)
	// os.Exit(0)

	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	loadedcfg, err := utils.NewConfig(cfg.configFile)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	err = validateCommand(cfg.command, loadedcfg)
	if err != nil {
		logger.Fatalf("Error validating command: %v", err)
	}

	issue := github.NewIssue(cfg.owner, cfg.repo, cfg.issueNumber, cfg.ghToken)

	userPrompt, imgs, err := constructUserPrompt(cfg.ghToken, issue, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Erro constructing userPrompt: %v", err)
	}

	prompt, err := constructPrompt(cfg.command, cfg.intent, userPrompt, imgs, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Error constructing prompt: %v", err)
	}

	aic, err := getAIClient(cfg.oaiKey, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Error geting AI client: %v", err)
	}

	emb, err := getEmbeddingClient(cfg.oaiKey, loadedcfg, logger)
	if err != nil {
		logger.Fatalf("Error geting AI client: %v", err)
	}
	fmt.Println("Prompt:", prompt.UserPrompt)
	doc, err := retriever.Retrieve(emb, prompt.UserPrompt, rag.Options{})
	// doc, err := retriever.Retrieve(emb, "What is Memory Management?", rag.Options{})
	if err != nil {
		fmt.Println("Error retrieving document:", err)
	}
	fmt.Println("Document:", doc)
	for _, d := range doc {
		prompt.UserPrompt += "\n" + d.String()
	}
	fmt.Println("Prompt:", prompt.UserPrompt)
	// os.Exit(0)

	comment, err := aic.GetResponse(prompt)
	if err != nil {
		logger.Fatalf("Error getting Response: %v", err)
	}
	logger.Println("Response:", comment)

	if err := issue.PostComment(comment); err != nil {
		logger.Fatalf("Error creating comment: %v", err)
	}
}

// Validate the provided command
func validateCommand(command string, cfg *utils.Config) error {
	if _, ok := cfg.Ai.Commands[command]; !ok {
		allowedCommands := make([]string, 0, len(cfg.Ai.Commands))
		for cmd := range cfg.Ai.Commands {
			allowedCommands = append(allowedCommands, cmd)
		}
		return fmt.Errorf("Invalid command: %s. Allowed commands are %s", command, strings.Join(allowedCommands, ", "))
	}
	return nil
}

// Construct user prompt from issue
func constructUserPrompt(ghToken string, issue *github.GitHubIssue, cfg *utils.Config, logger *log.Logger) (string, []ai.Image, error) {
	title, err := issue.GetTitle()
	if err != nil {
		return "", nil, fmt.Errorf("Error getting Title: %w", err)
	}

	body, err := issue.GetBody()
	if err != nil {
		return "", nil, fmt.Errorf("Error getting Body: %w", err)
	}

	var userPrompt strings.Builder
	userPrompt.WriteString("Title:" + *title + "\n")
	userPrompt.WriteString("Body:" + *body + "\n")

	comments, err := issue.GetComments()
	if err != nil {
		return "", nil, fmt.Errorf("Error getting comments: %w", err)
	}

	var images []ai.Image
	imageRegex := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)

	for _, v := range comments {
		if *v.User.Login == "github-actions[bot]" {
			continue
		}
		if cfg.System.Debug.Log_level == "debug" {
			logger.Printf("%s: %s", *v.User.Login, *v.Body)
		}
		userPrompt.WriteString(*v.User.Login + ":" + *v.Body + "\n")

		matches := imageRegex.FindAllStringSubmatch(*v.Body, -1)
		for _, match := range matches {
			logger.Println("Image URL:", match[2]) // Log the URL of the image
			imgData, ext, err := utils.DownloadImage(match[2], ghToken)
			if err != nil {
				return "", nil, fmt.Errorf("Error downloading image: %w", err)
			}

			images = append(images, ai.Image{Data: imgData, Extension: ext})
		}
	}
	return userPrompt.String(), images, nil
}

// Construct AI prompt
func constructPrompt(command, intent, userPrompt string, imgs []ai.Image, cfg *utils.Config, logger *log.Logger) (*ai.Prompt, error) {
	var systemPrompt string
	if command == "ask" {
		if intent == "" {
			return nil, fmt.Errorf("Error: intent is required for 'ask' command")
		}
		systemPrompt = cfg.Ai.Commands[command].System_prompt + intent + "\n"
	} else {
		systemPrompt = cfg.Ai.Commands[command].System_prompt
	}
	logger.Println("\x1b[34mPrompt: |\n", systemPrompt, userPrompt, "\x1b[0m")
	return &ai.Prompt{UserPrompt: userPrompt, SystemPrompt: systemPrompt, Images: imgs}, nil
}

// Initialize AI client
func getAIClient(oaiKey string, cfg *utils.Config, logger *log.Logger) (ai.Ai, error) {
	switch cfg.Ai.Provider {
	case "openai":
		if oaiKey == "" {
			return nil, fmt.Errorf("Error: Please provide your Open AI API key")
		}
		logger.Println("Using OpenAI API")
		logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
		return ai.NewOpenAIClient(oaiKey, cfg.Ai.OpenAI.Model), nil
	case "vertexai":
		logger.Println("Using VertexAI API")
		logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
		aic, err := ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
		if err != nil {
			return nil, fmt.Errorf("Error: new Vertex AI client: %w", err)
		}
		return aic, nil
	default:
		return nil, fmt.Errorf("Error: Invalid provider")
	}
}

// Initialize EmbeddingModel
func getEmbeddingClient(oaiKey string, cfg *utils.Config, logger *log.Logger) (ai.EmbeddingModel, error) {
	switch cfg.Ai.Provider {
	case "openai":
		if oaiKey == "" {
			return nil, fmt.Errorf("Error: Please provide your Open AI API key")
		}
		logger.Println("Using OpenAI API")
		logger.Println("OpenAI model:", cfg.Ai.OpenAI.Model)
		return ai.NewOpenAIClient(oaiKey, cfg.Ai.OpenAI.Model), nil
	case "vertexai":
		logger.Println("Using VertexAI API")
		logger.Println("VertexAI model:", cfg.Ai.VertexAI.Model)
		aic, err := ai.NewVertexAIClient(cfg.Ai.VertexAI.Project, cfg.Ai.VertexAI.Region, cfg.Ai.VertexAI.Model)
		if err != nil {
			return nil, fmt.Errorf("Error: new Vertex AI client: %w", err)
		}
		return aic, nil
	default:
		return nil, fmt.Errorf("Error: Invalid provider")
	}
}

// Initialize Neo4jRetriever
func getNeo4jRetriever(cfg *Neo4jConfig, logger *log.Logger) (*rag.Neo4jRetriever, error) {
	r, err := rag.NewNeo4jRetriever(cfg.uri, cfg.username, cfg.password, cfg.fulltextIndex, cfg.vectorIndex)
	if err != nil {
		return nil, fmt.Errorf("Error: new Neo4jRetriever: %w", err)
	}
	return r, nil
}
