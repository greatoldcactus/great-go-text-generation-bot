package chat_api

import "errors"

type ChatMode int

const (
	ModeChat = iota
	ModeSingleMessage
	ModeContinue
)

var ErrUnknownMode = errors.New("mode unknown")

type UserData struct {
	History       History  `json:"history"`
	Mode          ChatMode `json:"mode"`
	MaxTokens     uint     `json:"max_tokens"`
	NameCharacter string   `json:"name_character"`
	NameMe        string   `json:"name_me"`
	Model         string   `json:"model"`
}

func NewUser(mode ChatMode) (data UserData, err error) {
	data = UserData{}
	err = data.SetMode(mode)

	return
}

func (d *UserData) SetMode(mode ChatMode) (err error) {
	switch mode {
	case ModeChat, ModeContinue, ModeSingleMessage:
		d.Mode = mode
	default:
		err = ErrUnknownMode
	}

	return
}
