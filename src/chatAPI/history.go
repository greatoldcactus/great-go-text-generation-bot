package chat_api

import (
	"fmt"
	"strings"
)

type historyEntry struct {
	Character string `json:"role"`
	Message   string `json:"content"`
}

func (he *historyEntry) String() string {
	if he.Character == "" {
		return fmt.Sprintf("%s \n", he.Message)
	}
	return fmt.Sprintf("%s: \n %s \n", he.Character, he.Message)
}

type History struct {
	Data []historyEntry
}

func (h *History) String() string {
	sb := strings.Builder{}

	for _, he := range h.Data {
		sb.Write([]byte(he.String()))
	}

	return sb.String()
}

func (h *History) Clear() {
	h.Data = []historyEntry{}
}

func (h *History) AddMessage(character string, message string) {
	h.Data = append(h.Data, historyEntry{Character: character, Message: message})
}

func (h *History) GetData() []historyEntry {
	return h.Data
}

func NewChatHistory() History {
	return History{}
}
