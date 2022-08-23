package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"telegrammBot/cons"
	"telegrammBot/internal/handlers"
	"telegrammBot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/joho/godotenv"
)

var (
	keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Остатки"),
			tgbotapi.NewKeyboardButton("Перемещения"),
			//tgbotapi.NewKeyboardButton("Пустая1"),
		),
		//tgbotapi.NewKeyboardButtonRow(
		//	tgbotapi.NewKeyboardButton("Пустая2"),
		//	tgbotapi.NewKeyboardButton("Пустая3"),
		//	tgbotapi.NewKeyboardButton("Пустая4"),
		//),

	)

	msgToUser       string
	buttonRemainder = "Остатки"
	remainder models.Remainder

)

func main() {

	err := godotenv.Load("app.env")
	if err != nil {
		log.Fatalf("Error loading .env file: ")
	}

	// port, err := strconv.Atoi(os.Getenv("PORT"))
	// if err != nil {
	// 	port = 8081
	// }

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	webHookInfo := tgbotapi.NewWebhookWithCert(fmt.Sprintf("https://%s:%s/%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT"), bot.Token), cons.CERT_PAHT)

	_, err = bot.SetWebhook(webHookInfo)
	if err != nil {
		log.Fatal(err)
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		log.Fatal(err)
	}
	if info.LastErrorDate != 0 {
		log.Printf("Telegram callback failed: %s", info.LastErrorMessage)
	}
	updates := bot.ListenForWebhook("/" + bot.Token)

	// router := mux.NewRouter()

	// handler := webserver.NewHandler()
	// handler.Register(router)

	log.Println(fmt.Printf("Starting API server on %s:%s\n", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")))

	// if err := http.ListenAndServe(fmt.Sprintf("%s:%s", "95.52.245.250", os.Getenv("BOT_PORT")), router); err != nil {
	// 	log.Fatal(err)
	// }

	//log.Fatal(http.ListenAndServeTLS("0.0.0.0:8443", cons.CERT_PAHT, cons.KEY_PATH, nil))

	//log.Fatal(http.ListenAndServeTLS(fmt.Sprintf("%s:%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")), cons.CERT_PAHT, cons.KEY_PATH, nil))
	//go http.ListenAndServeTLS(fmt.Sprintf("%s:%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")), cons.CERT_PAHT, cons.KEY_PATH, nil)
	go http.ListenAndServeTLS("0.0.0.0:8443", cons.CERT_PAHT, cons.KEY_PATH, nil)

	for update := range updates {

		if update.Message == nil { // ignore non-Message updates
			continue
		}

		//fmt.Printf("Получено сообщение от пользователя: %+v\n", update.Message.Text)
		// log.Printf("LOG:%+v\n", update)
		// fmt.Printf("%v\n", "New update")
		// fmt.Printf("%v\n", update.Message)

		switch update.Message.Text {

		case "Остатки":

			err, remainderList := handlers.RemainderHandler()

			if err != nil {
				log.Printf("%+v\n", err.Error())
				msgToUser = err.Error()
			} else {

				i := 0
				body := make([]string, i)
				lenBody := make(map[int]int, i)

				for i <= len(remainderList)-1 {
					remainder = remainderList[i]
					body = append(body, fmt.Sprintf("Номенклатура: %v, Код номенклатуры: %v, Склад: %v", remainder.Nomenclature, remainder.Code, remainder.Store))
					msgToUser = strings.Join(body, "\n")

					lenBody[i] = len(msgToUser)
					i++
				}

				totalLengMsg := len(msgToUser)

				if totalLengMsg > cons.MaxLengMsg {
					max := totalLengMsg / cons.MaxLengMsg

					if (totalLengMsg % cons.MaxLengMsg) > 0 {
						max++
					}

					count := 1
					j := 0
					start := 0

					for count <= max {

						for i := j; i <= (totalLengMsg-1) && lenBody[i] <= cons.MaxLengMsg*count; i++ {
							j = i
						}

						end := lenBody[j]

						if count == max {
							end = totalLengMsg
						}

						err := sentToTelegramm(bot, update.Message.Chat.ID, msgToUser[start:end], buttonRemainder)

						start = lenBody[j]

						if err != nil {
							log.Printf("Error sending to user: %+v\n", err.Error())
							break
						}

						count++
					}

				} else {
					err := sentToTelegramm(bot, update.Message.Chat.ID, msgToUser, buttonRemainder)

					if err != nil {
						log.Printf("Error sending to user: %+v\n", err.Error())
						break
					}
				}

			}

		case "Перемещения":
			continue
		default:
			msgToUser = update.Message.Text
		}

	}

}

func sentToTelegramm(bot *tgbotapi.BotAPI, id int64, message string, button string) error {

	if button == buttonRemainder {

		msg := tgbotapi.NewMessage(id, message)
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
			return err
		}

	}

	return nil

}
