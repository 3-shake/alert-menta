package ai

type Ai interface {
	GetResponse(prompt string) (string, error)
}
