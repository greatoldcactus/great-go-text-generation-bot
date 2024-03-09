package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	chat_api "chat_api"

	tgbotapi "github.com/Syfaro/telegram-bot-api"
)

var (
	// глобальная переменная в которой храним токен
	telegramBotToken string
	config           Config
)

func init() {
	// принимаем на входе флаг -telegrambottoken
	flag.StringVar(&telegramBotToken, "telegrambottoken", "", "Telegram Bot Token")
	flag.Parse()

	// без него не запускаемся
	if telegramBotToken == "" {
		file, err := os.Open("telegram.token")
		if err == nil {
			token, err := io.ReadAll(file)
			if err == nil {
				telegramBotToken = string(token)
			} else {
				panic(err)
			}
		} else {
			panic(err)
		}
	}

	if telegramBotToken == "" {
		log.Print("-telegrambottoken is required")
		panic("telegram token undefined")
	}
}

func init() {
	file, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	string_config, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(string_config, &config)
	if err != nil {
		panic(err)
	}
}

type GenerationData struct {
	Msg       string
	User_data *UserData
}

var (
	GenerationChannel = make(chan GenerationData)
	MessageChan       = make(chan tgbotapi.MessageConfig)
)

func MessagingLoop(bot *tgbotapi.BotAPI) {
	for msg := range MessageChan {
		_, err := bot.Send(msg)
		if err != nil {
			err_text := fmt.Sprintf("error happened when trying to seng message: %s\n", err.Error())
			fmt.Println(err_text)
			msg := tgbotapi.NewMessage(msg.ChatID, err_text)
			bot.Send(msg)
		}
	}
}

func GeneratingLoop(bot *tgbotapi.BotAPI) {
	for gen_data := range GenerationChannel {
		result, err := Generate(gen_data.Msg, &gen_data.User_data.UserData)
		gen_data.User_data.StoreSimple()
		if err != nil {
			msg := tgbotapi.NewMessage(gen_data.User_data.UserID, fmt.Sprintf("error happened on generation %s", err.Error()))
			MessageChan <- msg
		} else {
			msg := tgbotapi.NewMessage(gen_data.User_data.UserID, result)
			MessageChan <- msg
		}

	}
}

func List_models() (msg string) {
	api, err := chat_api.NewSimpleApiRequest(config.Url, nil)
	if err != nil {
		msg = fmt.Sprintf("unable to create bot api: %v", err)
		return
	}

	models, err := api.ListModels()
	if err != nil {
		msg = fmt.Sprintf("unable to list models: %v", err)
		return
	}
	msg = "models: \n"

	for _, model := range models {
		msg = msg + "\n" + model
	}

	return
}

func Generate(msg string, user_data *chat_api.UserData) (result string, err error) {
	var api chat_api.SimpleApiRequest
	api, err = chat_api.NewSimpleApiRequest(config.Url, user_data)
	if err != nil {
		return
	}

	api.AddMyMessage(msg)
	result, err = api.Generate()

	return
}

func CreateSelectKeyboard(bot *tgbotapi.BotAPI, userdata *UserData) {
	api, err := chat_api.NewSimpleApiRequest(config.Url, nil)
	if err != nil {
		msg := tgbotapi.NewMessage(userdata.UserID, fmt.Sprintf("failed to init api for list models: %v", err))
		MessageChan <- msg
		return
	}
	models, err := api.ListModels()
	if err != nil {
		msg := tgbotapi.NewMessage(userdata.UserID, fmt.Sprintf("failed to list models: %v", err))
		MessageChan <- msg
		return
	}

	msg := tgbotapi.NewMessage(userdata.UserID, "choose model: ")

	//TODO pagination?
	buttons := [][]tgbotapi.InlineKeyboardButton{}
	for _, model := range models {
		data_string := fmt.Sprintf("model!%s", model)
		button := tgbotapi.NewInlineKeyboardButtonData(model, data_string)
		row := tgbotapi.NewInlineKeyboardRow(button)

		buttons = append(buttons, row)

	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(buttons...)

	msg.ReplyMarkup = keyboard

	MessageChan <- msg

}

func main() {
	// используя токен создаем новый инстанс бота
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// u - структура с конфигом для получения апдейтов
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// используя конфиг u создаем канал в который будут прилетать новые сообщения
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	go MessagingLoop(bot)
	go GeneratingLoop(bot)

	// в канал updates прилетают структуры типа Update
	// вычитываем их и обрабатываем
	for update := range updates {
		// универсальный ответ на любое сообщение
		reply := "Не знаю что сказать"

		if update.Message == nil {

			if update.CallbackQuery == nil {
				continue
			}
			go func() {
				data := update.CallbackQuery.Data
				callback_parts := strings.Split(data, "!")
				if len(callback_parts) < 1 {
					fmt.Println("error happened when trying to process callback: unable to detect type")
					return
				}
				callback_type := callback_parts[0]

				callback_payload := strings.TrimPrefix(data, fmt.Sprintf("%s!", callback_type))

				switch callback_type {
				case "model":
					user, err := GetUser(update.CallbackQuery.From.UserName, int64(update.CallbackQuery.From.ID))
					if err != nil {
						fmt.Println("failed to get user to set model ", err)
						return
					}
					user.Model = callback_payload
					user.StoreSimple()
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("model selected: %s", callback_payload))
					MessageChan <- msg

				}
			}()
			continue

		}

		chat := update.Message.Chat
		_, err := GetUser(chat.UserName, chat.ID)
		if err != nil {
			_, err := NewUser(chat.UserName, chat.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "failed to create new user")
				MessageChan <- msg
				continue
			}
		}

		// логируем от кого какое сообщение пришло
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		// свитч на обработку комманд
		// комманда - сообщение, начинающееся с "/"
		switch update.Message.Command() {
		case "start":
			reply = "hello there!"

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
			MessageChan <- msg
		case "help":
			reply = `
/start - start dialog
/help  - show this message
/list  - list models
/model - select model
/profile - get user data
/name_me - set my name for assistant
/name_assistant - set name for assistant
/clear - clear history
/tokens - sets token cnt
			`

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
			MessageChan <- msg
		case "list":
			go func() {
				reply := List_models()
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
				MessageChan <- msg
			}()
		case "model":
			user, err := GetUser(chat.UserName, chat.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("failed to find user: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			go CreateSelectKeyboard(bot, user)
		case "name_me":
			user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user data: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			new_name := update.Message.CommandArguments()
			if new_name == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "add name after command")
				MessageChan <- msg
				continue
			}
			user.NameMe = new_name
			user.StoreSimple()
		case "name_assistant":
			user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user data: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			new_name := update.Message.CommandArguments()
			if new_name == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "add name after command")
				MessageChan <- msg
				continue
			}
			user.NameCharacter = new_name
			user.StoreSimple()
		case "tokens":
			user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user data: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			new_token_cnt := update.Message.CommandArguments()
			if new_token_cnt == "" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "add name after command")
				MessageChan <- msg
				continue
			}
			cnt, err := strconv.Atoi(new_token_cnt)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("failed to parse size: '%s'", new_token_cnt))
				MessageChan <- msg
				continue
			}
			user.MaxTokens = uint(cnt)
			user.StoreSimple()
		case "profile":
			user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
			fmt.Println(user.History.Data)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user data: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			user_data, err := json.Marshal(user)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to serialize user data: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("user data: \n%s", user_data))
			MessageChan <- msg
		case "clear":
			user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
			fmt.Println(user.History.Data)
			if err != nil {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user dada for clear: %s", err.Error()))
				MessageChan <- msg
				continue
			}
			user.History.Clear()
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "history cleared")
			MessageChan <- msg
			user.StoreSimple()

		case "":
			go func() {
				user, err := GetUser(update.Message.Chat.UserName, update.Message.Chat.ID)
				fmt.Println(user.History.Data)
				if err != nil {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("unable to get user data: %s", err.Error()))
					MessageChan <- msg
					return
				}

				GenerationChannel <- GenerationData{
					Msg:       update.Message.Text,
					User_data: user,
				}
			}()

		default:
			reply = fmt.Sprintf("unknown command: %s", update.Message.Command())
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
			MessageChan <- msg
		}
	}
}
