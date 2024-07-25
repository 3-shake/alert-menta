package ai

type Ai interface {
	GetResponse(prompt Prompt) (string, error)
}

type Image struct {
	Data      []byte
	Extension string
}

type Prompt struct {
	UserPrompt   string
	SystemPrompt string
	Images       []Image
}
