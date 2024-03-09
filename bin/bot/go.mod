module text-generation-bot

go 1.22.0

require github.com/Syfaro/telegram-bot-api v4.6.4+incompatible

require (
	chat_api v0.0.0-00010101000000-000000000000 // indirect
	github.com/technoweenie/multipartstreamer v1.0.1 // indirect
)

replace chat_api => ../../src/chatAPI
