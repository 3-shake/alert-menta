package rag

import (
	"github.com/3-shake/alert-menta/internal/ai"
)

type Retriever interface {
	// Retrieve(ctx context.Context, query string, options ...Option) ([]Document, error)
	Retrieve(query string, embedding ai.EmbeddingModel, options Options) ([]Document, error)
	RetrieveByVector(vector []float32, options Options) ([]Document, error)
	RetrieveIssue(vector []float32, issueNumber uint32, options Options) string
}

type Options struct {
	TopK                  uint32
	Branches              []string
	WithStructuredData    bool // Not implemented yet
	EnableHybridRetrieval bool // Not implemented yet
}

type CodebaseEmbeddingOptions struct {
	Branches []string
}

type Document struct {
	Id      string
	Content string
	Branch  string
	URL     string
	Score   float64
}

func (d Document) String() string {
	str := "id: " + d.Id + ", content: " + d.Content
	return str
}
