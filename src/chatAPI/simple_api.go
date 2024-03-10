package chat_api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const average_token_len = 3

type SimpleApiRequest struct {
	BaseUrl string
	*UserData
}

func (sa *SimpleApiRequest) Connect() (err error) {
	return
}

var ErrBaseUrlNotDefined = errors.New("base URL not defined for API")
var ErrIncorrectResponse = errors.New("incorrect response")
var ErrIncorrectModel = errors.New("incorrect model")

func (sa *SimpleApiRequest) Generate() (result string, err error) {
	if sa.BaseUrl == "" {
		err = ErrBaseUrlNotDefined
		return
	}

	response_history := NewChatHistory()

	switch sa.Mode {
	case ModeChat:
		response_history = sa.History
	case ModeSingleMessage:
		data := sa.History.GetData()
		lastMessage := data[len(data)-1]
		response_history.AddMessage(lastMessage.Character, lastMessage.Message)
	case ModeContinue:
		data := sa.History.GetData()
		lastMessage := data[len(data)-1]
		response_history.AddMessage("Assistant", lastMessage.Message)

	default:
		err = ErrUnknownMode
		return
	}

	err = sa.CheckModel()
	if err != nil {
		return
	}

	type Options struct {
		Num_predict int `json:"num_predict"`
		Num_thread  int `json:"num_thread"`
	}

	tokens := sa.MaxTokens
	if tokens == 0 {
		tokens = 50
	}

	type history_message_node struct {
		Role    string `json:"role"`
		Message string `json:"content"`
	}

	final_history := []history_message_node{}
	for _, history_node := range response_history.Data {
		final_history = append(final_history, history_message_node{history_node.Character, history_node.Message})
	}

	payload := struct {
		Model    string                 `json:"model"`
		Messages []history_message_node `json:"messages"`
		Stream   bool                   `json:"stream"`
		Options  Options                `json:"options"`
	}{
		Model:    sa.Model,
		Messages: final_history,
		Stream:   false,
		Options: Options{
			Num_predict: int(tokens),
			Num_thread:  8,
		},
	}

	var json_payload []byte
	json_payload, err = json.Marshal(payload)
	if err != nil {
		return
	}

	url := sa.BaseUrl + "/api/chat"

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(json_payload)))

	if err != nil {
		return "", fmt.Errorf("unable to create request %w", err)
	}

	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {

		err := fmt.Errorf("incorrect response from server: %d", resp.StatusCode)
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", fmt.Errorf("unable to read body after request: %v, error: %w", resp.Body, err)
	}

	var response struct {
		Response string `json:"response"`
	}

	err = json.Unmarshal(body, &response)

	if err != nil {
		err = fmt.Errorf("unable to unmarshall response body %w", err)
		return
	}

	text_answer := response.Response

	switch sa.Mode {
	case ModeChat:
		sa.History.AddMessage(sa.NameCharacter, text_answer)
	case ModeSingleMessage:
		sa.History.Clear()
	case ModeContinue:
		sa.History.Clear()
		data := sa.History.GetData()
		lastMessage := data[len(data)-1]
		text := lastMessage.Message
		text = fmt.Sprintf("%s\n%s", text, text_answer)
		max_text_len := int(average_token_len * sa.MaxTokens)
		if len(text) > max_text_len {
			text = text[len(text)-max_text_len:]
		}
		sa.History.AddMessage("", text)

	default:
		return "", ErrUnknownMode
	}

	return text_answer, nil

}

func (sa *SimpleApiRequest) SetModel(model_name string) (err error) {
	sa.Model = model_name

	return nil
}

func (sa *SimpleApiRequest) CheckModel() error {
	models, err := sa.ListModels()
	if err != nil {
		return fmt.Errorf("unable to get model list from api %w", err)
	}

	model_ok := false

	for _, model := range models {
		if model == sa.Model {
			model_ok = true
			break
		}
	}

	if !model_ok {
		return fmt.Errorf("%w model: %s", ErrIncorrectModel, sa.Model)
	}

	return nil
}

func (sa *SimpleApiRequest) ListModels() (models []string, err error) {
	url := sa.BaseUrl + "/api/tags"

	resp, err := http.Get(url)

	if err != nil {
		err = fmt.Errorf("unable to GET %w", err)
		return
	}

	if resp.StatusCode != 200 {

		err = fmt.Errorf("incorrect response from server: %d", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		err = fmt.Errorf("unable to read from response body %w", err)
		return
	}

	var models_response struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	err = json.Unmarshal(body, &models_response)

	if err != nil {
		return
	}

	models = []string{}
	for _, model := range models_response.Models {
		models = append(models, model.Name)
	}

	return

}

func (sa *SimpleApiRequest) AddMyMessage(msg string) (err error) {
	return sa.AddMessage(sa.NameMe, msg)
}

func (sa *SimpleApiRequest) AddCharacterMessage(msg string) (err error) {
	return sa.AddMessage(sa.NameCharacter, msg)
}

func (sa *SimpleApiRequest) AddMessage(character string, msg string) (err error) {
	sa.History.AddMessage(character, msg)

	return
}

func NewSimpleApiRequest(url string, userdata *UserData) (api SimpleApiRequest, err error) {
	api = SimpleApiRequest{
		BaseUrl:  url,
		UserData: userdata,
	}

	return
}
