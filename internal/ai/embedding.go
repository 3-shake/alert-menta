package ai

type EmbeddingModel interface {
	GetEmbedding(string) ([]float32, error)
}
