package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"telegrammBot/cons"
	"telegrammBot/internal/handlers"
	"telegrammBot/internal/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/rs/zerolog"
	zrlog "github.com/rs/zerolog/log"

	"github.com/joho/godotenv"
)

var (
	keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Остатки"),
			tgbotapi.NewKeyboardButton("Перемещения"),
		),
	)

	msgToUser       string
	buttonRemainder = "Остатки"
	buttonMovements = "Перемещения"
	remainder       models.Remainder
)

func main() {

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	logFile, err := os.OpenFile("./temp/info.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	zrlog.Logger.Output(logFile)
	log.SetOutput(logFile)

	if err != nil {
		zrlog.Fatal().Msg(err.Error())
	}
	defer logFile.Close()

	err = godotenv.Load("app.env")
	if err != nil {
		zrlog.Fatal().Msg("Error loading .env file: ")
		log.Printf("FATAL: %s", "Error loading .env file: ")
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		log.Printf("FATAL: %v", err.Error())
	}

	bot.Debug = true

	zrlog.Info().Msg(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))
	log.Printf("%v", fmt.Sprintf("INFO: Authorized on account %s", bot.Self.UserName))

	webHookInfo := tgbotapi.NewWebhookWithCert(fmt.Sprintf("https://%s:%s/%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT"), bot.Token), cons.CERT_PAHT)

	_, err = bot.SetWebhook(webHookInfo)
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		log.Printf("FATAL: %v", err.Error())
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		log.Printf("FATAL: %v", err.Error())
	}
	if info.LastErrorDate != 0 {
		zrlog.Fatal().Msg(fmt.Sprintf("Telegram callback failed: %s", info.LastErrorMessage))
		log.Printf("FATAL: %v", fmt.Sprintf("Telegram callback failed: %s", info.LastErrorMessage))
	}
	updates := bot.ListenForWebhook("/" + bot.Token)

	//infoLog := log.New(logFile, fmt.Sprintf("INFO: Starting API server on %s:%s\n", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")), log.Ldate|log.Ltime)

	zrlog.Info().Msg(fmt.Sprintf("INFO: Starting API server on %s:%s\n", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")))
	log.Printf("INFO: %v", fmt.Sprintf("INFO: Starting API server on %s:%s\n", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")))

	go http.ListenAndServeTLS("0.0.0.0:8443", cons.CERT_PAHT, cons.KEY_PATH, nil)

	for update := range updates {

		if update.Message == nil { // ignore non-Message updates
			continue
		}

		fmt.Printf("Получено сообщение от пользователя: %+v\n", update.Message.Text)

		switch update.Message.Text {

		case "Перемещения":

			err, remainderList := handlers.MovementsHandler()

			if err != nil {
				zrlog.Fatal().Msg(err.Error())
				log.Printf("FATAL: %v", err.Error())
				msgToUser = err.Error()
			} else {

				sort.Sort(models.ArrayRemainder(remainderList))

				num := 1
				i := 0
				body := make([]string, i)
				lenBody := make(map[int]int, i)

				err := sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("*`-----` склад: \"%v\" `-----`*\n", remainderList[i].Store), lenBody, cons.StyleTextMarkdown, buttonMovements, "") //The first store

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					break
				}

				for i <= len(remainderList)-1 {

					st := remainderList[i].Store

					remainder = remainderList[i]
					body = append(body, fmt.Sprintf("%v", "___________________________________"))
					body = append(body, fmt.Sprintf("(%v). %s", num, remainder.Nomenclature))
					msgToUser = strings.Join(body, "\n")

					lenBody[i] = len(msgToUser)

					i++

					if i <= len(remainderList)-1 && st != remainderList[i].Store { //The store is turned change and expression "i <= len(remainderList)-1" still true.

						err := sentToTelegramm(bot, update.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextCommon, buttonMovements, st)

						body = nil
						num = 0

						//body = make([]string, 0)
						msgToUser = ""
						lenBody = nil
						//lenBody = make(map[int]int, 0)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							break
						}

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("*`----`склад: \"%v\"`----`*\n", remainderList[i].Store), lenBody, cons.StyleTextMarkdown, buttonMovements, "")

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							break
						}
					}

					num++
				}

				err = sentToTelegramm(bot, update.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, buttonMovements, remainderList[i-1].Store)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					break
				}

			}

		case "Остатки":

			err, remainderInformation := handlers.RemainderHandler()

			if err != nil {
				zrlog.Fatal().Msg(err.Error())
				log.Printf("FATAL: %v", err.Error())
				msgToUser = err.Error()
				break
			}

			err = sentToTelegramm(bot, update.Message.Chat.ID, remainderInformation.Information, nil, cons.StyleTextCommon, buttonRemainder, "")

			if err != nil {
				zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				break
			}

		default:
			msgToUser = update.Message.Text
		}

	}

}

func sentToTelegramm(bot *tgbotapi.BotAPI, id int64, message string, lenBody map[int]int, styleText string, button string, header string) error {

	totalLengMsg := len(message)

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

			formatMessage := message[start:end]

			if button == buttonMovements && header != "" {

				formatMessage = fmt.Sprintf("<i><b>%v</b></i>\n%v", header, formatMessage)

			}

			msg := tgbotapi.NewMessage(id, formatMessage, styleText)
			msg.ReplyMarkup = keyboard

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			start = lenBody[j]

			count++
		}

	} else {

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	}

	return nil

}
