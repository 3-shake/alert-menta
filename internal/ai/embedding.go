package ai

type EmbeddingModel interface {
	GetEmbedding(text string) ([]float32, error)
}
