package ai

type Ai interface {
	GetResponse(prompt *Prompt) (string, error)
}

type Prompt struct {
	UserPrompt   string
	SystemPrompt string
}
