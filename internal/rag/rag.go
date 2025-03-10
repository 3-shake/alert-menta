package rag

import (
	"github.com/3-shake/alert-menta/internal/ai"
)

type Retriever interface {
	// Retrieve(ctx context.Context, query string, options ...Option) ([]Document, error)
	Retrieve(query string, embedding ai.EmbeddingModel, options Options) ([]Document, error)
	RetrieveByVector(vector []float32, options Options) ([]Document, error)
}

type Options struct {
	topK                  uint32
	withStructuredData    bool // Not implemented yet
	enableHybridRetrieval bool // Not implemented yet
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
