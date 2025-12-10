package domain

// FoundString represents a string pattern that was found in a user message
type FoundString struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

// MessageResult represents the result of processing a user message
type MessageResult struct {
	User    User          `json:"user"`
	Matches []FoundString `json:"matches"`
}
