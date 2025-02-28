package rag

import (
	"github.com/3-shake/alert-menta/internal/ai"
)

type Retriever interface {
	// Retrieve(ctx context.Context, query string, options ...Option) ([]Document, error)
	Retrieve(embedding ai.EmbeddingModel, options Options) ([]Document, error)
}

type Options struct {
	topK                  int
	withStructuredData    bool
	enableHybridRetrieval bool
}

type Document struct {
	Id      string
	Content string
	Score   float64
}

func (d Document) String() string {
	str := "id: " + d.Id + ", content: " + d.Content
	return str
}
