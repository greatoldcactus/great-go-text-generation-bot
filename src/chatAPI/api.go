package chat_api

type API interface {
	Connect() error
	Generate() error
	AddMessage(character string, msg string) error
}
