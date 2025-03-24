package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

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
	pineconeKey string
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
	// neo4jcfg := &Neo4jConfig{}
	flag.StringVar(&cfg.repo, "repo", "", "Repository name")
	flag.StringVar(&cfg.owner, "owner", "", "Repository owner")
	flag.IntVar(&cfg.issueNumber, "issue", 0, "Issue number")
	flag.StringVar(&cfg.intent, "intent", "", "Question or intent for the 'ask' command")
	flag.StringVar(&cfg.command, "command", "", "Commands to be executed by AI. Commands defined in the configuration file are available.")
	flag.StringVar(&cfg.configFile, "config", "", "Configuration file")
	flag.StringVar(&cfg.ghToken, "github-token", "", "GitHub token")
	flag.StringVar(&cfg.oaiKey, "api-key", "", "OpenAI api key")
	flag.BoolVar(&cfg.useRag, "use-rag", false, "Use RAG model for response generation")
	flag.StringVar(&cfg.pineconeKey, "pinecone-api-key", "", "Pinecone api key")
	flag.Parse()

	if cfg.repo == "" || cfg.owner == "" || cfg.issueNumber == 0 || cfg.ghToken == "" || cfg.command == "" || cfg.configFile == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	var retriever rag.Retriever
	if cfg.useRag {
		flag.Parse()
		if cfg.pineconeKey == "" {
			log.Fatalf("Error: If -useRag is set, -pinecone-api-key is required")
		}
		r, err := getPineconeRetriever(cfg)
		retriever = r
		if err != nil {
			log.Fatalf("Error getting retriever: %v", err)
		}
	}

	logger := log.New(
		os.Stdout, "[alert-menta main] ",
		log.Ldate|log.Ltime|log.Llongfile|log.Lmsgprefix,
	)

	loadedcfg, err := utils.NewConfig(cfg.configFile)
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	if cfg.command == "upsert_db" {
		idxName := rag.GetPineconeIndexName(cfg.owner, cfg.repo)
		err := CreateDB(idxName, []string{"main"}, cfg, loadedcfg, logger)
		if err != nil {
			logger.Fatalf("Error upserting DB: %v", err)
		}
		return
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

	var docs []rag.Document
	relatedIssue := ""
	if cfg.useRag {
		emb, err := getEmbeddingClient(cfg.oaiKey, loadedcfg, logger)
		if err != nil {
			logger.Fatalf("Error geting AI client: %v", err)
		}

		if loadedcfg.Ai.Commands[cfg.command].Similar_code {
			ragPrompt, err := constructRAGPrompt(cfg.command, cfg.intent, userPrompt, imgs, *issue, loadedcfg, logger)
			if err != nil {
				logger.Fatalf("Error constructing RAG prompt: %v", err)
			}

			ragComment, err := aic.GetResponse(ragPrompt)
			if err != nil {
				logger.Fatalf("Error getting RAG comment: %v", err)
			}
			logger.Println("RAG Comment:", ragComment)

			ragVector, err := emb.GetEmbedding(ragComment)
			if err != nil {
				logger.Fatalf("Error getting RAG vector: %v", err)
			}

			docs, err = retriever.RetrieveByVector(ragVector, rag.Options{})
			for _, d := range docs {
				prompt.UserPrompt += "\n" + d.String()
			}
		}
		if loadedcfg.Ai.Commands[cfg.command].Similar_issue {
			issueVector, err := emb.GetEmbedding(userPrompt)
			if err != nil {
				logger.Fatalf("Error getting issue vector: %v", err)
			}
			relatedIssue = retriever.RetrieveIssue(issueVector, uint32(cfg.issueNumber), rag.Options{})
		}
	}

	comment, err := aic.GetResponse(prompt)
	if err != nil {
		logger.Fatalf("Error getting Response: %v", err)
	}
	if cfg.useRag {
		comment += "\n\n" + "## Sources:\n"
		for i, d := range docs {
			comment += fmt.Sprintf("%d. [%s](%s)\n", i+1, d.Id, d.URL)
		}
		comment += "\n\n" + relatedIssue
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

// RAG の前処理を行うプロンプトを作成する関数
func constructRAGPrompt(command, intent, userPrompt string, imgs []ai.Image, issue github.GitHubIssue, cfg *utils.Config, logger *log.Logger) (*ai.Prompt, error) {
	systemPrompt := `
I'm looking to identify related files and functions to solve a GitHub Issue. Please provide analysis and advice based on the information I'll share in the following format:

## Analysis Requested:
1. Files likely related to this Issue and why
2. Specific functions or code blocks that should be investigated
3. Possible root causes of the problem
4. Approaches for resolution

Please suggest specific file paths and function names where possible. Maximize the use of information available from the repository structure to understand the code architecture before making suggestions.
	`
	userPromptPlaceholder := `## GitHub Issue:
{{.UserPrompt}}

## Repository Structure:
{{.RepositoryStructure}}
	`
	userPromptTmpl, err := template.New("userPrompt").Parse(userPromptPlaceholder)
	if err != nil {
		logger.Fatalf("Error parsing userPrompt template: %v", err)
	}
	type PromptData struct {
		UserPrompt          string
		RepositoryStructure string
	}
	defaultBranch, _ := issue.GetDefaultBranch()
	lf, _ := issue.ListFiles(defaultBranch)
	lfs := strings.Join(lf, "\n")
	userPromptBuf := strings.Builder{}
	err = userPromptTmpl.Execute(&userPromptBuf, PromptData{UserPrompt: userPrompt, RepositoryStructure: lfs})
	userPrompt = userPromptBuf.String()
	logger.Println("\x1b[34mRAGPrompt: |\n", systemPrompt, userPrompt, "\x1b[0m")
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
	logger.Println("Neo4jRetriever:", r)
	if err != nil {
		return nil, fmt.Errorf("Error: new Neo4jRetriever: %w", err)
	}
	return r, nil
}

// Initialize PineconeRetriever
func getPineconeRetriever(cfg *Config) (*rag.PineconeClient, error) {
	idxName := rag.GetPineconeIndexName(cfg.owner, cfg.repo)
	r, err := rag.NewPineconeClient(idxName, cfg.pineconeKey)
	if err != nil {
		return nil, fmt.Errorf("Error: new Neo4jRetriever: %w", err)
	}
	return r, nil
}

func CreateDB(idxName string, targetBranches []string, cfg *Config, loadedcfg *utils.Config, logger *log.Logger) error {
	logger.Println("Creating DB to Index:", idxName)
	repoURL := fmt.Sprintf("https://github.com/%s/%s", cfg.owner, cfg.repo)
	repo, err := utils.CloneRepository(repoURL, &utils.AuthOptions{Username: cfg.owner, Token: cfg.ghToken})
	if err != nil {
		return fmt.Errorf("Error cloning repository: %w", err)
	}
	branches, err := utils.GetBranches(repo, []string{})
	_, err = utils.ListFiles(repo)

	var docs []rag.Document
	for _, branch := range branches {
		branchDocs, err := rag.ConvertBranchtoDocuments(cfg.owner, cfg.repo, repo, branch)
		if err != nil {
			return fmt.Errorf("Error converting branch to documents: %w", err)
		}
		docs = append(docs, *branchDocs...)
	}

	emb, err := getEmbeddingClient(cfg.oaiKey, loadedcfg, logger)
	if err != nil {
		return fmt.Errorf("Error getting embedding client: %w", err)
	}

	pc, err := rag.NewPineconeClient(idxName, cfg.pineconeKey)
	if err != nil {
		return fmt.Errorf("Error getting Pinecone client: %w", err)
	}

	pc.CreateCodebaseDB(docs, emb, rag.CodebaseEmbeddingOptions{Branches: targetBranches})

	issues := github.GetAllIssues(cfg.owner, cfg.repo, cfg.ghToken)
	pc.CreateIssueDB(issues, emb)

	return nil
}
