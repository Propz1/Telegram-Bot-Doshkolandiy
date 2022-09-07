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
	"github.com/signintech/gopdf"

	"github.com/joho/godotenv"
)

var (
	keyboard = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Остатки"),
			tgbotapi.NewKeyboardButton("Перемещения"),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Перемещения PDF"),
		),
	)

	msgToUser          string
	buttonRemainder    = "Остатки"
	buttonMovements    = "Перемещения"
	buttonMovementsPDF = "Перемещения PDF"
	remainder          models.Remainder

	cellOption_Caption = gopdf.CellOption{Align: 16}
	cellOption_Default = gopdf.CellOption{Align: 8}
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

		case "/start":

			err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Привет %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, "", "")

			if err != nil {
				zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
			}

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
					return
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

						err := sentToTelegramm(bot, update.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, buttonMovements, st)

						body = nil
						num = 0

						//body = make([]string, 0)
						msgToUser = ""
						lenBody = nil
						lenBody = make(map[int]int, 0)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("*`----`склад: \"%v\"`----`*\n", remainderList[i].Store), lenBody, cons.StyleTextMarkdown, buttonMovements, "")

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

					num++
				}

				err = sentToTelegramm(bot, update.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, buttonMovements, remainderList[i-1].Store)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					return
				}

			}

		case "Остатки":

			err, remainderInformation := handlers.RemainderHandler()

			if err != nil {
				zrlog.Fatal().Msg(err.Error())
				log.Printf("FATAL: %v", err.Error())
				msgToUser = err.Error()
				return
			}

			err = sentToTelegramm(bot, update.Message.Chat.ID, remainderInformation.Information, nil, cons.StyleTextCommon, buttonRemainder, "")

			if err != nil {
				zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				return
			}

		case "Перемещения PDF":

			err, remainderList := handlers.MovementsHandler()

			/////////////////////////////////////////////////////////////////////////////

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 1", Code: "1", Store: "Темный"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 2", Code: "2", Store: "Темный"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 111", Code: "1111", Store: "Дальний"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 1112", Code: "1112", Store: "Дальний"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 1113", Code: "1113", Store: "Дальний"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 888", Code: "888", Store: "Ближний"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 999", Code: "8898", Store: "Ближний"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи 986", Code: "88996", Store: "Ближний"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи ваыа4464654", Code: "888", Store: "Удаленный"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи вав454545", Code: "8898", Store: "Удаленный"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи ва5454646", Code: "88995", Store: "Удаленный"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи sfsdfа4464654", Code: "888", Store: "Башкирский"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи ваыа4464654", Code: "888", Store: "Башкирский"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Башкирский"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Выборгский"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddfddffdfdf546464664444464654", Code: "888", Store: "Астрaханский"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Галицкий"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Тверской"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Ульяновский"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf446ва4654", Code: "888", Store: "Ульяновский"})

			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddf4464654", Code: "888", Store: "Узбекский"})
			// remainderList = append(remainderList, models.Remainder{Nomenclature: "Устройство связи dfdffdfddfsd4464654", Code: "888", Store: "Узбекский"})
			///////////////////////////////////////////////////////////////////////////////

			if err != nil {
				zrlog.Fatal().Msg(err.Error())
				log.Printf("FATAL: %v", err.Error())
				msgToUser = err.Error()

			} else {

				sort.Sort(models.ArrayRemainder(remainderList))

				pdf := gopdf.GoPdf{}
				pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

				err = pdf.AddTTFFont("a_AlternaNr", "./external/fonts/ttf/a_AlternaNr.ttf")

				if err != nil {
					log.Print(err.Error())
					return
				}

				err = pdf.AddTTFFont("Inter-ExtraLight", "./external/fonts/ttf/Inter-ExtraLight.ttf")

				if err != nil {
					log.Print(err.Error())
					return
				}

				err = pdf.AddTTFFont("Inter-Bold", "./external/fonts/ttf/Inter-Bold.ttf")

				if err != nil {
					log.Print(err.Error())
					return
				}

				err = pdf.AddTTFFont("Merriweather-Bold", "./external/fonts/ttf/Merriweather-Bold.ttf")

				if err != nil {
					log.Print(err.Error())
					return
				}

				var capacityLine int = 42

				num := 1
				i := 0
				y := 15.0
				line := 0
				page := 0

				for i <= len(remainderList)-1 {

					if line >= capacityLine || page == 0 {

						pdf.AddPage()
						line = 1
						page++

						y = 15.0

						pdf.SetXY(570, y)
						pdf.SetTextColorCMYK(100, 100, 100, 100)
						err = pdf.SetFont("a_AlternaNr", "", 10)
						if err != nil {
							log.Print(err.Error())
							return
						}
						err = pdf.Text(fmt.Sprintf("стр %v", page))
						if err != nil {
							log.Print(err.Error())
							return
						}
						line++

						y = 20.0

						pdf.SetXY(260, y)
						y = 60
						pdf.SetTextColorCMYK(0, 100, 100, 0)
						err = pdf.SetFont("Merriweather-Bold", "", 14)
						if err != nil {
							log.Print(err.Error())
							return
						}
						err = pdf.CellWithOption(nil, remainderList[i].Store, cellOption_Caption)
						if err != nil {
							log.Print(err.Error())
							return
						}
						line = line + 2

					}

					pdf.SetTextColorCMYK(100, 100, 100, 100)
					err = pdf.SetFont("Inter-ExtraLight", "", 12)
					if err != nil {
						log.Print(err.Error())
						return
					}

					st := remainderList[i].Store

					remainder = remainderList[i]

					pdf.SetXY(10, y)
					y = y + 20
					err = pdf.Text(fmt.Sprintf("(%v). %s", num, remainder.Nomenclature))
					if err != nil {
						log.Print(err.Error())
						return
					}
					line++

					i++

					if i <= len(remainderList)-1 && st != remainderList[i].Store { //The store is turned change and expression "i <= len(remainderList)-1" still true.
						num = 0

						pdf.SetXY(260, y)
						y = y + 40
						pdf.SetTextColorCMYK(0, 100, 100, 0)
						err = pdf.SetFont("Merriweather-Bold", "", 14)
						if err != nil {
							log.Print(err.Error())
							return
						}
						err = pdf.CellWithOption(nil, remainderList[i].Store, cellOption_Caption)
						if err != nil {
							log.Print(err.Error())
							return
						}
						line = line + 2

					}

					num++
				}

				err = pdf.WritePdf("./external/files/Movements.pdf")

				if err != nil {
					log.Print(err.Error())
					return
				}

				// err = pdf.Image("./imgs/test.jpg", 0.5, 0.5, nil) //print image
				// if err != nil {
				// 	log.Print(err.Error())
				// 	return
				// }

				err = sentToTelegrammPDF(bot, update.Message.Chat.ID, fmt.Sprintf("./external/files/%s.pdf", "Movements"), "")

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending file pdf to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending file pdf to user: %+v\n", err.Error()))
					return
				}

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

		start := 0
		end := totalLengMsg

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

	}

	return nil

}

func sentToTelegrammPDF(bot *tgbotapi.BotAPI, id int64, pdf_path string, file_id string) error {

	var msg tgbotapi.DocumentConfig

	if file_id != "" {
		msg = tgbotapi.NewDocumentShare(id, file_id)
	} else {
		msg = tgbotapi.NewDocumentUpload(id, pdf_path)
	}

	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		zrlog.Panic().Msg(err.Error())
		log.Printf("PANIC: %v", err.Error())
		return err
	}

	return nil
}
