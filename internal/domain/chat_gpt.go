package domain

type ChatGPTResponse struct {
	Choices []ChatGPTChoice `json:"choices,omitempty"`
}

type ChatGPTChoice struct {
	Message ChatGPTMessage `json:"message"`
}

type ChatGPTMessage struct {
	Content string `json:"content,omitempty"`
}
