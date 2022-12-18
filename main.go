package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"telegrammBot/cons"
	"telegrammBot/internal/botcommand"
	"telegrammBot/internal/botstate"
	"telegrammBot/internal/cache"
	"telegrammBot/internal/enumapplic"
	"telegrammBot/internal/models"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	pgxpool "github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	zrlog "github.com/rs/zerolog/log"
	"github.com/signintech/gopdf"
	"gopkg.in/gomail.v2"

	"github.com/joho/godotenv"
)

var (
	keyboardMainMenue = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Заполнить заявку"),
			tgbotapi.NewKeyboardButton("Отправить работу"),
		),

		// tgbotapi.NewKeyboardButtonRow(
		// 	tgbotapi.NewKeyboardButton("Остатки"),
		// 	tgbotapi.NewKeyboardButton("Перемещения PDF"),
		// ),
	)

	keyboardApplicationStart = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Продолжить"),
			tgbotapi.NewKeyboardButton("Отмена"),
		),
	)

	keyboardContinueDataPolling1 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отменить заявку"),
		),
	)

	keyboardContinueDataPolling2 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Далее"),
			tgbotapi.NewKeyboardButton("Отменить заявку"),
		),
	)

	keyboardConfirm = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Подтвердить"),
			tgbotapi.NewKeyboardButton("Исправить"),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Отменить заявку"),
		),
	)

	keyboardAdmin = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Оценить работу"),
			tgbotapi.NewKeyboardButton("Публикация в VK"),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Настройки"),
		),
	)

	contests           = [4]string{"CONTEST1", "CONTEST2", "CONTEST3", "CONTEST4"}
	userPolling        = cache.NewCacheDataPolling()
	msgToUser          string
	buttonRemainder    = "Остатки"
	buttonMovements    = "Перемещения"
	buttonMovementsPDF = "Перемещения PDF"
	remainder          models.Remainder
	requestWarehouses  = "Выберите склад"
	botsCommand        = [10]string{"CompletedApplication", "SendPublication", "Movements", "MovementsPDF", "/start", "EnterPassword", "Settings", "AppendUser", "ShowUsers", "DeleteUser"}

	cellOption_Caption = gopdf.CellOption{Align: 16}
	cellOption_Default = gopdf.CellOption{Align: 8}

	maxWidthPDF = 507.0

	cacheBotSt cache.CacheBotSt
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

	cacheBotSt = cache.NewCacheBotSt()

	for update := range updates {

		if update.Message == nil && update.CallbackQuery == nil { // ignore non-Message updates and no CallbackQuery
			continue
		}

		if update.Message != nil {

			if update.Message.Photo != nil {

				fmt.Printf("Получено фото!\n\n\n")

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_PHOTO.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FILE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				} else if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_FILE || cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_FILE_CORRECTION {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_FILE.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO_CORRECTION {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_PHOTO.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			}

			if update.Message.Document != nil {

				ph := *update.Message.Document

				go getFile(bot, update.Message.Chat.ID, ph.FileID, *userPolling, botstate.ASK_FILE.EnumIndex())

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					return
				}

			}

			messageByteText := bytes.TrimPrefix([]byte(update.Message.Text), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			messageText := string(messageByteText[:])

			//fmt.Printf("%v\n", messageText)

			switch messageText {

			case "/start":

				err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Здравствуйте, %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.START, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

			case botcommand.COMPLETE_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PROJECT)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите конкурс:", nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION, "", nil, "", false)

				//err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Здесь будет краткая инструкция по заполнению заявки.....  %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION.String(), "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.SELECT_PROJECT.String():

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PROJECT {

					if userPolling.Get(update.Message.Chat.ID).Agree {

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP)

					} else {

						err = sentToTelegramm(bot, update.Message.Chat.ID, "Для продолжения необходимо дать согласние на обработку персональных данных. Или нажмите \"Отмена\"", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				}

			case botcommand.CANCEL.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", nil, "", false)

				//err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Здесь будет краткая инструкция по заполнению заявки.....  %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION.String(), "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.CANCEL_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", nil, "", false)

				//err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Здесь будет краткая инструкция по заполнению заявки.....  %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION.String(), "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.START_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP)

				err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.START_APPLICATION, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case "Настройки":

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите действие:", nil, cons.StyleTextCommon, botcommand.SETTINGS, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.SETTINGS)

			default:

				stateBot := cacheBotSt.Get(update.Message.Chat.ID)

				switch stateBot {

				case botstate.ASK_FNP:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_AGE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_FNP_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_AGE:

					age, err := strconv.Atoi(messageText)

					if err != nil {
						// zrlog.Fatal().Msg(fmt.Sprintf("Error convert age: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error convert age: %+v\n", err.Error()))

						err := sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите, пожалуйста, возраст в правильном формате (цифрой/цифрами):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					} else if age > 120 || age == 0 || age < 0 {

						err := sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Пожалуйста, укажите \"реальный возраст\" (цифрой/цифрами):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				case botstate.ASK_AGE_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAME_INSTITUTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_LOCALITY)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAME_INSTITUTION_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_LOCALITY:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAMING_UNIT)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_LOCALITY_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAMING_UNIT:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAMING_UNIT_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_PUBLICATION_TITLE:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP_LEADER)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (нажать \"Далее\" если нет руководителя):", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_FNP_LEADER, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_PUBLICATION_TITLE_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_FNP_LEADER:

					if messageText != botcommand.DOWN.String() {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, "")
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				case botstate.ASK_FNP_LEADER_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_EMAIL:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_EMAIL_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_CHECK_DATA:

					if messageText == botcommand.SELECT_CORRECTION.String() {

						cacheBotSt.Set(update.Message.Chat.ID, botstate.SELECT_CORRECTION)

						err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите пункт который нужно исправить:", nil, cons.StyleTextCommon, botcommand.SELECT_CORRECTION, "", nil, "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else if messageText == "Подтвердить" {

						if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_CHECK_DATA {

							err := sentToTelegramm(bot, update.Message.Chat.ID, "Регистрирую...", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", nil, "", false)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								return
							}

							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Unable to establish connection to database: %+v\n", err.Error()))
								log.Printf("FATAL: %v", fmt.Sprintf("Unable to establish connection to database: %+v\n", err.Error()))
								os.Exit(1)
								return
							}
							defer dbpool.Close()

							dbpool.Config().MaxConns = 7

							err = AppendRequisition(update.Message.Chat.ID, dbpool, ctx)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Error append requisition to db: %+v\n", err.Error()))
								log.Printf("FATAL: %v", fmt.Sprintf("Error append requisition to db: %+v\n", err.Error()))
								os.Exit(1)
								return
							}

							ok, err := ConvertRequisitionToPDF(update.Message.Chat.ID)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Error converting requisition into PDF: %+v\n", err.Error()))
								log.Printf("FATAL: %v", fmt.Sprintf("Error converting requisition into PDF: %+v\n", err.Error()))
								fmt.Printf(err.Error())
							}

							if !ok {
								// Отправляем просто в тексте
								fmt.Printf("Не ОК")

							} else {

								numReq := userPolling.Get(update.Message.Chat.ID).RequisitionNumber
								path_reqPDF := fmt.Sprintf("./external/files/Заявка_№%v.pdf", numReq)

								userPolling.Set(update.Message.Chat.ID, enumapplic.REQUISITION_PDF, path_reqPDF)

								err = sentToTelegrammPDF(bot, update.Message.Chat.ID, path_reqPDF, "", botcommand.UNDEFINED)

								if err != nil {
									zrlog.Fatal().Msg(fmt.Sprintf("Error sending file pdf to user: %v\n", err))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending file pdf to user: %v\n", err))
									return
								}

								// Отправляем на почту
								t := time.Now()
								formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

								send, err := SentEmail(os.Getenv("ADMIN_EMAIL"), update.Message.Chat.ID, true, fmt.Sprintf("Заявка №%v от %s (%s)", userPolling.Get(update.Message.Chat.ID).RequisitionNumber, formattedTime, userPolling.Get(update.Message.Chat.ID).DocumentType), "", "")

								if err != nil {

									zrlog.Fatal().Msg(fmt.Sprintf("Error sending letter to admin's email: %+v\n", err.Error()))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending letter to admin's email: %+v\n", err.Error()))

									fmt.Printf("%v", err)

								}

								if send {

									go deleteUserPolling(update.Message.Chat.ID, *userPolling)

								}

								err = sentToTelegramm(bot, update.Message.Chat.ID, "Поздравляем, Ваша заявка зарегестрирована!", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", nil, "", false)

								if err != nil {
									zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
									return
								}

							}

						}

					} else if messageText == "Отменить заявку" {
						userPolling.Delete(update.Message.Chat.ID)

					} else {
						//Сообщаем пользователю, что бы нажал одну из кнопок меню.
					}

				case botstate.UNDEFINED:
					msgToUser = update.Message.Text
				}

			}
		}

		if update.CallbackQuery != nil {

			callbackQueryData := bytes.TrimPrefix([]byte(update.CallbackQuery.Data), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			callbackQueryText := string(callbackQueryData[:])

			// var warehouse string
			// var PDF bool

			switch callbackQueryText {

			case contests[0]:

			case contests[1]:

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST2)

				//Concise description of contest
				description := GetConciseDescription(contests[1])

				err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case contests[2]:

			case contests[3]:

			case cons.CERTIFICATE.String():

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DOCUMENT_TYPE, string(cons.CERTIFICATE))
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.CERTIFICATE.String())

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				} else {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case cons.DIPLOMA.String():

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DOCUMENT_TYPE, string(cons.DIPLOMA))
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.DIPLOMA.String())

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				} else {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS1:

				cb := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

				if cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS || cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS1)

					//Меняем проверку данных на вопрос прикрепить фото работы

					//cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)
					//err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			// case cons.PLACE_DELIVERY_OF_DOCUMENTS2:

			// 	cb := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

			// 	if cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS || cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

			// 		userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS2)
			// 		cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

			// 		err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

			// 		if err != nil {
			// 			zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
			// 			log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
			// 			return
			// 		}

			// 	}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS3:

				cb := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

				if cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS || cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS3)
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.CANSEL_CORRECTION.String(): //CallBackQwery "CANSEL_CORRECTION"
				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.FNP.String(): //CallBackQwery "FNP"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.START_APPLICATION, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					}

				}

			case enumapplic.AGE.String(): //CallBackQwery "AGE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.NAME_INSTITUTION.String(): //CallBackQwery "NAME_INSTITUTION"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.LOCALITY.String(): //CallBackQwery "LOCALITY"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_LOCALITY_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.NAMING_UNIT.String(): //CallBackQwery "NAMING_UNIT"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAMING_UNIT_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.PUBLICATION_TITLE.String(): //CallBackQwery "PUBLICATION_TITLE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case enumapplic.FNP_LEADER.String(): //CallBackQwery "FNP_LEADER"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_LEADER_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя:", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.EMAIL.String(): //CallBackQwery "EMAIL"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_EMAIL_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.DOCUMENT_TYPE.String(): //CallBackQwery "DOCUMENT_TYPE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String(): //CallBackQwery "PLACE_DELIVERY_OF_DOCUMENTS"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case enumapplic.PHOTO.String(): //CallBackQwery "PHOTO"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.FILE.String(): //CallBackQwery "FILE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FILE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", nil, "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case cons.AGREE.String():

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Agree, "")

				err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Согласие на обработку персональных данных получено", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", nil, "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					return
				}

			}

			// if ok := strings.Contains(callbackQueryText, "RemainderRequest_"); ok {

			// 	if ok := strings.Contains(callbackQueryText, "_PDF"); ok {
			// 		PDF = true
			// 		warehouse = callbackQueryText[17 : len(callbackQueryText)-4]
			// 	} else {
			// 		PDF = false
			// 		warehouse = callbackQueryText[17:]
			// 	}

			// 	fmt.Printf("warehouse:%v\n\n", warehouse)

			// 	switch PDF {

			// 	case true:

			// 		var position models.RemainderQuantity

			// 		warehouseRemainder, err := handlers.RemainderHandler(warehouse)

			// 		remListWarehouse := warehouseRemainder.RemainderList

			// 		sort.Sort(models.RemainderList(remListWarehouse))

			// 		if err != nil {
			// 			zrlog.Fatal().Msg(fmt.Sprintf("Error getting response from web-service 1C: %+v\n", err.Error()))
			// 			log.Printf("FATAL: %v", fmt.Sprintf("Error getting response from web-service 1C: %+v\n", err.Error()))
			// 			msgToUser = err.Error()
			// 		} else {

			// 			pdf := gopdf.GoPdf{}
			// 			pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

			// 			err := pdf.AddTTFFont("a_AlternaNr", "./external/fonts/ttf/a_AlternaNr.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("Inter-ExtraLight", "./external/fonts/ttf/Inter-ExtraLight.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("Inter-Bold", "./external/fonts/ttf/Inter-Bold.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("Merriweather-Bold", "./external/fonts/ttf/Merriweather-Bold.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("Inter-ExtraLight", "./external/fonts/ttf/Merriweather-Bold.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("arialblack", "./external/fonts/ttf/arialblack.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			err = pdf.AddTTFFont("times", "./external/fonts/ttf/times.ttf")

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			var capacityLine int = 41

			// 			num := 1
			// 			i := 0
			// 			step := 15.0
			// 			line := 0
			// 			page := 0
			// 			var vertLine_y float64

			// 			for i <= len(remListWarehouse)-1 {

			// 				if line >= capacityLine || page == 0 {

			// 					pdf.AddPage()
			// 					line = 1
			// 					page = pdf.GetNumberOfPages()

			// 					step = 15.0

			// 					vertLine_y = 18

			// 					pdf.SetLineWidth(0.2)
			// 					pdf.Line(10, step+3, 581, step+3) //horizontal

			// 					pdf.SetXY(560, step)
			// 					pdf.SetTextColorCMYK(100, 100, 100, 100)
			// 					err := pdf.SetFont("a_AlternaNr", "", 10)
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}
			// 					err = pdf.Text(fmt.Sprintf("стр %v", page))
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}
			// 					line++

			// 					step = 20.0

			// 					pdf.SetXY(260, step)
			// 					step = 60
			// 					pdf.SetTextColorCMYK(0, 100, 100, 0)
			// 					err = pdf.SetFont("Merriweather-Bold", "", 14)
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}
			// 					err = pdf.CellWithOption(nil, warehouse, cellOption_Caption)
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}

			// 					pdf.Line(10, 40, 581, 40) //horizontal

			// 					pdf.Line(10, vertLine_y, 10, step-20)      //vertical 1
			// 					pdf.Line(34, vertLine_y+22, 34, step-20)   //vertical 2
			// 					pdf.Line(542, vertLine_y+22, 542, step-20) //vertical 3
			// 					pdf.Line(581, vertLine_y, 581, step-20)    //vertical 4

			// 					vertLine_y = step - 20

			// 					line++
			// 				}

			// 				pdf.SetTextColorCMYK(100, 100, 100, 100)
			// 				// err = pdf.SetFont("arialblack", "", 12)
			// 				// if err != nil {
			// 				// 	log.Print(err.Error())
			// 				// }
			// 				err = pdf.SetFont("times", "", 12)
			// 				if err != nil {
			// 					log.Print(err.Error())
			// 				}

			// 				position = remListWarehouse[i]

			// 				if len(strconv.Itoa(num)) == 1 {
			// 					pdf.SetX(19)
			// 				} else if len(strconv.Itoa(num)) == 2 {
			// 					pdf.SetX(16)
			// 				} else {
			// 					pdf.SetX(12)
			// 				}
			// 				pdf.SetY(step)
			// 				step = step + 20

			// 				err = pdf.SetFont("Inter-ExtraLight", "", 12)
			// 				if err != nil {
			// 					log.Print(err.Error())
			// 				}
			// 				err = pdf.Text(fmt.Sprintf("%v", num))
			// 				if err != nil {
			// 					log.Print(err.Error())
			// 				}

			// 				// err = pdf.SetFont("arialblack", "", 12)
			// 				// if err != nil {
			// 				// 	log.Print(err.Error())
			// 				// }
			// 				err = pdf.SetFont("times", "", 12)
			// 				if err != nil {
			// 					log.Print(err.Error())
			// 				}

			// 				pdf.SetX(37)
			// 				text := strings.TrimSpace(position.Nomenclature)
			// 				widthText, err := pdf.MeasureTextWidth(text)
			// 				if err != nil {
			// 					log.Print(err.Error())
			// 				}

			// 				if widthText > maxWidthPDF {

			// 					var arrayText []string

			// 					arrayText, err = pdf.SplitText(text, maxWidthPDF)
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}

			// 					for i, t := range arrayText {

			// 						pdf.SetX(36)
			// 						if i == 0 {
			// 							pdf.Line(10, vertLine_y, 10, step-2)   //vertical 1
			// 							pdf.Line(34, vertLine_y, 34, step-2)   //vertical 2
			// 							pdf.Line(542, vertLine_y, 542, step-2) //vertical 3
			// 							pdf.Line(581, vertLine_y, 581, step-2) //vertical 4

			// 							err = pdf.Text(t)
			// 							step = step + 5
			// 						}

			// 						if err != nil {
			// 							log.Print(err.Error())
			// 						}

			// 						if i == 1 {
			// 							pdf.SetY(step - 10)

			// 							err = pdf.Text(t)
			// 							if err != nil {
			// 								log.Print(err.Error())
			// 							}

			// 							pdf.SetX(543)
			// 							err = pdf.Text(fmt.Sprintf("%v", position.Quantity))
			// 							if err != nil {
			// 								log.Print(err.Error())
			// 							}

			// 							pdf.Line(10, step-7, 581, step-7) //horizontal
			// 							step = step + 5
			// 						}

			// 						line++
			// 					}

			// 				} else {

			// 					y := pdf.GetY()

			// 					err = pdf.Text(text)
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}

			// 					pdf.SetX(543)
			// 					err = pdf.Text(fmt.Sprintf("%v", position.Quantity))
			// 					if err != nil {
			// 						log.Print(err.Error())
			// 					}

			// 					pdf.Line(10, y+3, 581, y+3) //horizontal

			// 					pdf.Line(10, vertLine_y, 10, y+3)   //vertical 1
			// 					pdf.Line(34, vertLine_y, 34, y+3)   //vertical 2
			// 					pdf.Line(542, vertLine_y, 542, y+3) //vertical 3
			// 					pdf.Line(581, vertLine_y, 581, y+3) //vertical 4

			// 					line++

			// 				}
			// 				i++
			// 				num++
			// 			}

			// 			err = pdf.WritePdf(fmt.Sprintf("./external/files/Quantity_%s.pdf", warehouse))

			// 			if err != nil {
			// 				log.Print(err.Error())
			// 			}

			// 			// err = pdf.Image("./imgs/test.jpg", 0.5, 0.5, nil) //print image
			// 			// if err != nil {
			// 			// 	log.Print(err.Error())
			// 			// 	return
			// 			// }

			// 			err = sentToTelegrammPDF(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("./external/files/Quantity_%s.pdf", warehouse), "")

			// 			if err != nil {
			// 				zrlog.Fatal().Msg(fmt.Sprintf("Error sending file pdf to user: %v\n", err))
			// 				log.Printf("FATAL: %v", fmt.Sprintf("Error sending file pdf to user: %v\n", err))
			// 				return
			// 			}

			// 		}

			// 	case false:

			// 		warehouseRemainder, err := handlers.RemainderHandler(warehouse)

			// 		remListWarehouse := warehouseRemainder.RemainderList

			// 		if err != nil {
			// 			zrlog.Fatal().Msg(fmt.Sprintf("Error getting response from web-service 1C: %v\n", err))
			// 			log.Printf("FATAL: %v", fmt.Sprintf("Error getting response from web-service 1C: %v\n", err))
			// 			msgToUser = err.Error()
			// 		} else {

			// 			num := 1
			// 			i := 0
			// 			body := make([]string, i)
			// 			lenBody := make(map[int]int, i)

			// 			for i <= len(remListWarehouse)-1 {

			// 				infoQuantity := remListWarehouse[i]

			// 				body = append(body, fmt.Sprintf("%v", "___________________________________"))
			// 				body = append(body, fmt.Sprintf("(%v). %s (%s) <b>%v шт</b>", num, infoQuantity.Nomenclature, infoQuantity.Code, infoQuantity.Quantity))
			// 				msgToUser = strings.Join(body, "\n")

			// 				lenBody[i] = len(msgToUser)

			// 				i++
			// 				num++
			// 			}

			// 			err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, botsCommand[1], "", nil, fmt.Sprintf("Остатки склада: \"%v\"", warehouse), false)

			// 			if err != nil {
			// 				zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %v\n", err))
			// 				log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %v\n", err))
			// 				return
			// 			}

			// 		}
			// 	}

			// } else if callbackQueryText == botsCommand[7] {

			// 	lenBody := make(map[int]int, 0)
			// 	msgToUser = ""

			// 	msgToUser = "Введите через пробел id пользователя, имя пользователя и телефон пользователя (опционально)"

			// 	err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, botsCommand[7], "", nil, "", false)

			// 	if err != nil {
			// 		zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %v\n", err))
			// 		log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %v\n", err))
			// 		return
			// 	}

			// } else if callbackQueryText == botsCommand[8] {

			// 	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))

			// 	if err != nil {
			// 		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
			// 		zrlog.Fatal().Msg(fmt.Sprintf("Unable to connect to database:: %v\n", err))
			// 		log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Unable to connect to database: %v\n", err))
			// 		os.Exit(1)
			// 	}
			// 	defer dbpool.Close()

			// 	rows, err := dbpool.Query(context.Background(), "SELECT userid, username, phone FROM users")

			// 	if err != nil {
			// 		zrlog.Fatal().Msg(fmt.Sprintf("Query to db is failed: %v\n", err))
			// 		log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Query to db is failed: %v\n", err))
			// 		os.Exit(1)
			// 	}

			// 	num := 1
			// 	i := 0
			// 	body := make([]string, i)
			// 	lenBody := make(map[int]int, i)
			// 	msgToUser = ""

			// 	// iterate through the rows
			// 	for rows.Next() {

			// 		values, err := rows.Values()
			// 		if err != nil {
			// 			log.Fatal("error while iterating dataset")
			// 		}

			// 		// convert DB types to Go types
			// 		userid := values[0].(int32)
			// 		username := values[1].(string)
			// 		phone := values[2].(string)

			// 		body = append(body, fmt.Sprintf("%v", "------------------------------------------------------"))
			// 		body = append(body, fmt.Sprintf("(%v). id %v, name %s, phone %s", num, userid, username, phone))
			// 		msgToUser = strings.Join(body, "\n")

			// 		lenBody[i] = len(msgToUser)

			// 		i++
			// 		num++

			// 		//fmt.Printf("id %v, username %s, phone %s", userid, username, phone)
			// 		//fmt.Printf("msgToUser %s", msgToUser)

			// 	}

			// 	err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, msgToUser, lenBody, cons.StyleTextHTML, botsCommand[8], "", nil, "", false)

			// 	if err != nil {
			// 		zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
			// 		log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
			// 		return
			// 	}

			// } else if callbackQueryText == botsCommand[9] {

			// }
		}
	}

}

func sentToTelegramm(bot *tgbotapi.BotAPI, id int64, message string, lenBody map[int]int, styleText string, command botcommand.BotCommand, button string, buttons []models.Warehouse, header string, PDF bool) error {

	switch command {

	case botcommand.SELECT_CORRECTION:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton4 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton5 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton6 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton7 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton8 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton9 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton10 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton11 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton12 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton13 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()), enumapplic.FNP.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()), enumapplic.AGE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()), enumapplic.NAME_INSTITUTION.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()), enumapplic.LOCALITY.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton4)

			inlineKeyboardButton5 = append(inlineKeyboardButton5, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()), enumapplic.NAMING_UNIT.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton5)

			inlineKeyboardButton6 = append(inlineKeyboardButton6, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()), enumapplic.PUBLICATION_TITLE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton6)

			inlineKeyboardButton7 = append(inlineKeyboardButton7, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()), enumapplic.FNP_LEADER.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton7)

			inlineKeyboardButton8 = append(inlineKeyboardButton8, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()), enumapplic.EMAIL.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton8)

			inlineKeyboardButton9 = append(inlineKeyboardButton9, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()), enumapplic.DOCUMENT_TYPE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton9)

			inlineKeyboardButton10 = append(inlineKeyboardButton10, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton10)

			inlineKeyboardButton11 = append(inlineKeyboardButton11, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.PHOTO.EnumIndex(), enumapplic.PHOTO.String()), enumapplic.PHOTO.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton11)

			inlineKeyboardButton12 = append(inlineKeyboardButton12, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()), enumapplic.FILE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton12)

			// inlineKeyboardButton11 = append(inlineKeyboardButton11, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s", enumapplic.CANSEL_CORRECTION.String()), enumapplic.CANSEL_CORRECTION.String()))
			// rowsButton = append(rowsButton, inlineKeyboardButton11)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			message = "или"
			msg = tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = keyboardContinueDataPolling1

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			message = "нажмите"
			msg = tgbotapi.NewMessage(id, message, styleText)

			rowsButton = nil
			inlineKeyboardButton13 = append(inlineKeyboardButton13, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s", enumapplic.CANSEL_CORRECTION.String()), enumapplic.CANSEL_CORRECTION.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton13)
			inlineKeyboardMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

		}

	case botcommand.START:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.COMPLETE_APPLICATION:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton4 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.CONTEST1), "CONTEST1"))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(string(cons.CONTEST2), "CONTEST2"))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(string(cons.CONTEST3), "CONTEST3"))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(string(cons.CONTEST4), "CONTEST4"))
			rowsButton = append(rowsButton, inlineKeyboardButton4)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}
		}

	case botcommand.SELECT_PROJECT:

		if !thisIsAdmin(id) {

			msg := tgbotapi.NewMessage(id, message, styleText) //Описание (инструкция) подачи заявки для участия в выбранном проекте

			msg.ReplyMarkup = keyboardApplicationStart

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			msg = tgbotapi.NewMessage(id, "или здесь", styleText)

			msg.ReplyMarkup = keyboardApplicationStart

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			err := sentToTelegrammPDF(bot, id, "./external/files/Положение ТК Синичка невеличка и ее друзья.pdf", "", botcommand.SELECT_PROJECT)

			if err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

			body := make([]string, 3)
			body = append(body, "В любой момент вы можете отменить заявку, нажав \"Отмена\"")
			body = append(body, "")
			body = append(body, "Для продолжения заполнения заявки, необходимо дать согласие на обработку персональных данных и нажать \"Продолжить\".\n Ознакомиться с пользователським соглашением и политикой конфидециальности\n можно по ссылке https://vk.com/topic-138597952_49458742 ")
			text := strings.Join(body, "\n")

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.AGREE), cons.AGREE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg = tgbotapi.NewMessage(id, text, cons.StyleTextCommon)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

		}

	case botcommand.WAITING_FOR_ACCEPTANCE:

		if !thisIsAdmin(id) {

			msg := tgbotapi.NewMessage(id, message, cons.StyleTextCommon)

			if !userPolling.Get(id).Agree {
				var rowsButton [][]tgbotapi.InlineKeyboardButton

				inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
				inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.AGREE), cons.AGREE.String()))
				rowsButton = append(rowsButton, inlineKeyboardButton1)
				inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

				msg.ReplyMarkup = inlineKeyboardMarkup
			} else {
				msg.ReplyMarkup = keyboardApplicationStart
			}

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}

		}

	case botcommand.CANCEL:

		msg := tgbotapi.NewMessage(id, message, styleText) //Выход в главное меню

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.CONTINUE_DATA_POLLING:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling1
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.RECORD_TO_DB:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling1
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SELECT_FNP_LEADER:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling2
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SELECT_DOCUMENT_TYPE:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.CERTIFICATE), cons.CERTIFICATE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(string(cons.DIPLOMA), cons.DIPLOMA.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}
		}

	case botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			//inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.PLACE_DELIVERY_OF_DOCUMENTS1, cons.PLACE_DELIVERY_OF_DOCUMENTS1))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			// inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.PLACE_DELIVERY_OF_DOCUMENTS2, cons.PLACE_DELIVERY_OF_DOCUMENTS2))
			// rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.PLACE_DELIVERY_OF_DOCUMENTS3, cons.PLACE_DELIVERY_OF_DOCUMENTS3))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}
		}

	case botcommand.CHECK_DATA:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardConfirm
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

		message = UserDataToStringForTelegramm(id)

		msg = tgbotapi.NewMessage(id, message, cons.StyleTextHTML)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardConfirm
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SETTINGS:

		if thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData("Добавить пользователя", "AppendUser"))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData("Показать всех пользователей бота", "ShowUsers"))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData("Удалить пользователя по ID", "DeleteUser"))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				zrlog.Panic().Msg(err.Error())
				log.Printf("PANIC: %v", err.Error())
				return err
			}
		}

	}

	return nil

}

func sentToTelegrammPDF(bot *tgbotapi.BotAPI, id int64, pdf_path string, file_id string, command botcommand.BotCommand) error {

	var msg tgbotapi.DocumentConfig

	switch command {

	case botcommand.SELECT_PROJECT:

		if file_id != "" {
			msg = tgbotapi.NewDocumentShare(id, file_id)
		} else {
			msg = tgbotapi.NewDocumentUpload(id, pdf_path)
		}

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardApplicationStart
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	default:

		if file_id != "" {
			msg = tgbotapi.NewDocumentShare(id, file_id)
		} else {
			msg = tgbotapi.NewDocumentUpload(id, pdf_path)
		}

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdmin
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	}

	return nil
}

func thisIsAdmin(id int64) bool {

	if i, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64); err == nil {
		log.Printf("strconv.ParseInt: ADMIN_ID=%d, type: %T\n", i, i)

		return id == i
	}

	return false
}

func parseUserData(messageText string) (error, []string) {

	usersDatas := strings.Split(messageText, "")

	// var userID int64
	// var username string
	// var phone string

	for k, v := range usersDatas {

		if k == 0 {
			_, err := strconv.Atoi(v)
			if err != nil {
				return fmt.Errorf("Bad user ID:%W", err), nil
			}
		}
	}

	return nil, usersDatas
}

func checkUserID(messageText string) bool {

	usersDatas := strings.Split(messageText, " ")

	_, err := strconv.Atoi(usersDatas[0])
	if err != nil {
		return false
	}

	return true

}

func AppendUser(usersDate []string) (error, bool) {

	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		zrlog.Fatal().Msg(fmt.Sprintf("Unable to connect to database:: %v\n", err))
		log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Unable to connect to database: %v\n", err))
		os.Exit(1)

		return err, false
	}
	defer dbpool.Close()

	_, err = dbpool.Query(context.Background(), "INSERT INTO users (userid, username, phone) VALUES ($1,$2,$3)", usersDate[0], usersDate[1], usersDate[2])

	if err != nil {
		zrlog.Fatal().Msg(fmt.Sprintf("Query to db is failed: %v\n", err))
		log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Query to db is failed: %v\n", err))
		os.Exit(1)

		return err, false
	}

	return nil, true

}

func AppendRequisition(userID int64, dbpool *pgxpool.Pool, ctx context.Context) error {

	userData := userPolling.Get(userID)

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	// defer cancel()

	// conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))

	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Unable to establish connection to database: %v\n", err)
	// 	zrlog.Fatal().Msg(fmt.Sprintf("Unable to establish connection to database: %v\n", err))
	// 	log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Unable to establish connection to database: %v\n", err))
	// 	os.Exit(1)

	// 	return err
	// }

	// _, err = conn.Exec(ctx, "insert into requisitions (user_id, contest, user_fnp, user_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, document_type, place_delivery_of_document) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) returning requisition_number", userID, userData.Contest, userData.FNP, userData.Age, userData.NameInstitution, userData.Locality, userData.NamingUnit, userData.PublicationTitle, userData.LeaderFNP, userData.Email, userData.DocumentType, userData.PlaceDeliveryDocuments)

	// if err != nil {
	// 	zrlog.Fatal().Msg(fmt.Sprintf("Query to db is failed: %v\n", err))
	// 	log.Printf("%v,%v", os.Stderr, fmt.Sprintf("Query to db is failed: %v\n", err))
	// 	os.Exit(1)

	// 	return err
	// }

	// return nil

	//dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

	// if err != nil {
	// 	return fmt.Errorf("Unable to establish connection to database: %w", err)
	// }
	// defer dbpool.Close()

	row, err := dbpool.Query(ctx, fmt.Sprintf("insert into %s (user_id, contest, user_fnp, user_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, document_type, place_delivery_of_document, start_date, expiration, close_date) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) returning requisition_number", userData.TableDB), userID, userData.Contest, userData.FNP, userData.Age, userData.NameInstitution, userData.Locality, userData.NamingUnit, userData.PublicationTitle, userData.LeaderFNP, userData.Email, userData.DocumentType, userData.PlaceDeliveryDocuments, time.Now().UnixNano(), int64(time.Now().Add(172800*time.Second).UnixNano()), 0)

	if err != nil {
		return fmt.Errorf("Query to db is failed: %W", err)
	}

	if row.Next() {

		var requisition_number int

		err := row.Scan(&requisition_number)

		if err != nil {
			return fmt.Errorf("Scan datas of row is failed %w", err)
		}

		userPolling.Set(userID, enumapplic.REQUISITION_NUMBER, fmt.Sprintf("%v", requisition_number))
	}

	return row.Err()
}

func ConvertRequisitionToPDF(userID int64) (bool, error) {

	usersRequisition := userPolling.Get(userID)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := pdf.AddTTFFont("a_AlternaNr", "./external/fonts/ttf/a_AlternaNr.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("Inter-ExtraLight", "./external/fonts/ttf/Inter-ExtraLight.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("Inter-Bold", "./external/fonts/ttf/Inter-Bold.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("Merriweather-Bold", "./external/fonts/ttf/Merriweather-Bold.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("Inter-ExtraLight", "./external/fonts/ttf/Merriweather-Bold.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("arialblack", "./external/fonts/ttf/arialblack.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.AddTTFFont("times", "./external/fonts/ttf/times.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	// step := 15.0
	// line := 0
	// page := 0
	//var vertLine_y float64

	pdf.SetTextColorCMYK(100, 100, 100, 100)

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	pdf.Image("./external/imgs/RequisitionsBoilerplate.jpg", 0, 0, rect)

	// line = 1
	// page = pdf.GetNumberOfPages()

	// step = 15.0

	//vertLine_y = 18

	//pdf.SetLineWidth(0.2)
	//pdf.Line(10, step+3, 581, step+3) //horizontal

	// pdf.SetXY(560, step)
	// pdf.SetTextColorCMYK(100, 100, 100, 100)
	// err = pdf.SetFont("a_AlternaNr", "", 10)
	// if err != nil {
	// 	log.Print(err.Error())
	// }
	// err = pdf.Text(fmt.Sprintf("стр %v", page))
	// if err != nil {
	// 	log.Print(err.Error())
	// }

	//step = 20.0

	pdf.SetXY(150, 220)
	pdf.SetTextColorCMYK(100, 70, 0, 67)
	err = pdf.SetFont("Merriweather-Bold", "", 14)
	if err != nil {
		log.Print(err.Error())
	}

	t := time.Now()
	formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

	err = pdf.CellWithOption(nil, fmt.Sprintf("Заявка №%v от %v ", usersRequisition.RequisitionNumber, formattedTime), cellOption_Caption)
	if err != nil {
		log.Print(err.Error())
	}

	y := 270.0
	step := 30.0

	pdf.SetXY(25, y)
	err = pdf.Text(fmt.Sprintf("Участник: %s", usersRequisition.FNP))
	if err != nil {
		log.Print(err.Error())
	}

	y = y + step
	pdf.SetXY(25, y)
	if usersRequisition.LeaderFNP != "" {
		err = pdf.Text(fmt.Sprintf("Руководитель: %s", usersRequisition.LeaderFNP))
		if err != nil {
			log.Print(err.Error())
		}
		y = y + step
	}

	pdf.SetXY(25, y)

	err = pdf.Text(fmt.Sprintf("Конкурс: \"%s\"", usersRequisition.Contest))
	if err != nil {
		log.Print(err.Error())
	}
	y = y + step

	pdf.SetXY(25, y)
	text := fmt.Sprintf("%s: \"%s\"", enumapplic.NAMING_UNIT, usersRequisition.NamingUnit)
	widthText, err := pdf.MeasureTextWidth(text)
	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			log.Print(err.Error())
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: \"%s\"", enumapplic.PUBLICATION_TITLE, usersRequisition.PublicationTitle)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			log.Print(err.Error())
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: %s", enumapplic.DOCUMENT_TYPE, usersRequisition.DocumentType)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			log.Print(err.Error())
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: %s", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, usersRequisition.PlaceDeliveryDocuments)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			log.Print(err.Error())
		}
		y = y + step
	}

	// pdf.Line(10, 40, 581, 40) //horizontal

	// pdf.Line(10, vertLine_y, 10, step-20)      //vertical 1
	// pdf.Line(34, vertLine_y+22, 34, step-20)   //vertical 2
	// pdf.Line(542, vertLine_y+22, 542, step-20) //vertical 3
	// pdf.Line(581, vertLine_y, 581, step-20)    //vertical 4

	//vertLine_y = step - 20

	//pdf.SetTextColorCMYK(100, 100, 100, 100)
	// 	// err = pdf.SetFont("arialblack", "", 12)
	// 	// if err != nil {
	// 	// 	log.Print(err.Error())
	// 	// }
	// 	err = pdf.SetFont("times", "", 12)
	// 	if err != nil {
	// 		log.Print(err.Error())
	// 	}

	// 	position = remListWarehouse[i]

	// 	if len(strconv.Itoa(num)) == 1 {
	// 		pdf.SetX(19)
	// 	} else if len(strconv.Itoa(num)) == 2 {
	// 		pdf.SetX(16)
	// 	} else {
	// 		pdf.SetX(12)
	// 	}
	// 	pdf.SetY(step)
	// 	step = step + 20

	// 	err = pdf.SetFont("Inter-ExtraLight", "", 12)
	// 	if err != nil {
	// 		log.Print(err.Error())
	// 	}
	// 	err = pdf.Text(fmt.Sprintf("%v", num))
	// 	if err != nil {
	// 		log.Print(err.Error())
	// 	}

	// 	// err = pdf.SetFont("arialblack", "", 12)
	// 	// if err != nil {
	// 	// 	log.Print(err.Error())
	// 	// }
	// 	err = pdf.SetFont("times", "", 12)
	// 	if err != nil {
	// 		log.Print(err.Error())
	// 	}

	// 	pdf.SetX(37)
	// 	text := strings.TrimSpace(position.Nomenclature)
	// 	widthText, err := pdf.MeasureTextWidth(text)
	// 	if err != nil {
	// 		log.Print(err.Error())
	// 	}

	// 	if widthText > maxWidthPDF {

	// 		var arrayText []string

	// 		arrayText, err = pdf.SplitText(text, maxWidthPDF)
	// 		if err != nil {
	// 			log.Print(err.Error())
	// 		}

	// 		for i, t := range arrayText {

	// 			pdf.SetX(36)
	// 			if i == 0 {
	// 				pdf.Line(10, vertLine_y, 10, step-2)   //vertical 1
	// 				pdf.Line(34, vertLine_y, 34, step-2)   //vertical 2
	// 				pdf.Line(542, vertLine_y, 542, step-2) //vertical 3
	// 				pdf.Line(581, vertLine_y, 581, step-2) //vertical 4

	// 				err = pdf.Text(t)
	// 				step = step + 5
	// 			}

	// 			if err != nil {
	// 				log.Print(err.Error())
	// 			}

	// 			if i == 1 {
	// 				pdf.SetY(step - 10)

	// 				err = pdf.Text(t)
	// 				if err != nil {
	// 					log.Print(err.Error())
	// 				}

	// 				pdf.SetX(543)
	// 				err = pdf.Text(fmt.Sprintf("%v", position.Quantity))
	// 				if err != nil {
	// 					log.Print(err.Error())
	// 				}

	// 				pdf.Line(10, step-7, 581, step-7) //horizontal
	// 				step = step + 5
	// 			}

	// 			line++
	// 		}

	// 	} else {

	// 		y := pdf.GetY()

	// 		err = pdf.Text(text)
	// 		if err != nil {
	// 			log.Print(err.Error())
	// 		}

	// 		pdf.SetX(543)
	// 		err = pdf.Text(fmt.Sprintf("%v", position.Quantity))
	// 		if err != nil {
	// 			log.Print(err.Error())
	// 		}

	// 		pdf.Line(10, y+3, 581, y+3) //horizontal

	// 		pdf.Line(10, vertLine_y, 10, y+3)   //vertical 1
	// 		pdf.Line(34, vertLine_y, 34, y+3)   //vertical 2
	// 		pdf.Line(542, vertLine_y, 542, y+3) //vertical 3
	// 		pdf.Line(581, vertLine_y, 581, y+3) //vertical 4

	// 		line++

	// 	}
	// 	i++
	// 	num++
	// }

	err = pdf.WritePdf(fmt.Sprintf("./external/files/Заявка_№%v.pdf", usersRequisition.RequisitionNumber))

	if err != nil {
		log.Print(err.Error())

		return false, err
	}

	return true, nil
}

func SentEmail(to string, userID int64, toAdmin bool, subject string, addFile string, message string) (bool, error) {

	if toAdmin {

		message = UserDataToString(userID)

	}

	m := gomail.NewMessage()

	m.SetHeader("From", os.Getenv("BOT_EMAIL"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.Embed(userPolling.Get(userID).Photo)
	m.Attach(userPolling.Get(userID).File)

	// Set the email body. You can set plain text or html with text/html
	m.SetBody("text/html", message)

	if addFile != "" {
		m.Attach(addFile)
	}

	// Settings for SMTP server
	d := gomail.NewDialer(os.Getenv("SMTP_SERVER"), 465, os.Getenv("BOT_LOGIN_EMAIL"), os.Getenv("BOT_PASSWORD_EMAIL"))

	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
		return false, err
	}

	return true, nil
}

func UserDataToString(userID int64) string {

	var text string

	usdata := userPolling.Get(userID)

	body := make([]string, 12)

	//<img src=\"./external/imgs/Titmouse.png\" width=\"300\" height=\"300\" alt=\"Синичка невеличка и ee друзья\" align=\"middle\">

	body = append(body, "<!DOCTYPE html><html lang=\"ru\"><body><dl>")

	body = append(body, "<style type=\"text/css\">BODY {margin: 0; /* Убираем отступы в браузере */}#toplayer {background: #F5FFFA; /* Цвет фона */height: 800px /* Высота слоя */}</style>")

	//body = append(body, fmt.Sprintf("<p>%v</p>", "___________________________________"))
	body = append(body, fmt.Sprintf("<div id=\"toplayer\"><dt><p><b>(%v). %s:</b></p></dt>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.Contest))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("<p>%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", usdata.FNP))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %v</p></dd>", usdata.Age))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.NameInstitution))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", usdata.Locality))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.NamingUnit))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.PublicationTitle))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	if usdata.LeaderFNP == "" {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v).</b> <s><i><b>%s:</b></i></s></p></dt>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", "-"))
	} else {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.LeaderFNP))
	}
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.Email))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.DocumentType))
	text = strings.Join(body, "\n")

	//body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.PlaceDeliveryDocuments))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.PHOTO.EnumIndex(), enumapplic.PHOTO.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", "Прикреплена"))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd></div>", "Прикреплена"))
	text = strings.Join(body, "\n")

	body = append(body, "</dl></body></html>")
	text = strings.Join(body, "\n")

	return text
}

func UserDataToStringForTelegramm(userID int64) string {

	var text string

	usdata := userPolling.Get(userID)

	body := make([]string, 12)

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.Contest))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.FNP))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))
	body = append(body, fmt.Sprintf("      %v", usdata.Age))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.NameInstitution))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.Locality))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.NamingUnit))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.PublicationTitle))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	if usdata.LeaderFNP == "" {
		body = append(body, fmt.Sprintf("(%v). <s><i><b>%s:</b></i></s>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("      %s", "-"))
	} else {
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.LeaderFNP))
	}
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.Email))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.DocumentType))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
	body = append(body, fmt.Sprintf("      %s", usdata.PlaceDeliveryDocuments))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PHOTO.EnumIndex(), enumapplic.PHOTO.String()))
	body = append(body, fmt.Sprintf("      %s(%s)", "Прикреплена", usdata.Photo))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("%v", "___________________________________"))
	body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()))
	body = append(body, fmt.Sprintf("      %s(%s)", "Прикреплена", usdata.File))
	text = strings.Join(body, "\n")

	return text
}

func GetConciseDescription(contest string) string {

	var text string

	body := make([]string, 14)

	if contest == contests[1] {

		body = append(body, "<b>В заявке потребуется указать следующие данные:\n</b>")
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.CONTEST.EnumIndex(), enumapplic.FNP.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
		body = append(body, fmt.Sprintf("(13). <b>%s</b>", "Фото работы"))
		body = append(body, fmt.Sprintf("(14). <b>%s</b>", "Квитанцию об оплате организационного взноса"))
		body = append(body, "\n")
		body = append(body, "Подробнее с условиями конкурса можно ознакомиться на сайте https://vk.com/topic-138597952_49394008\n")
		body = append(body, "\n")

		text = strings.Join(body, "\n")
	}

	return text
}

func downloadFile(filepath string, url string) (err error) {

	// Create the file

	out, err := os.Create(filepath)

	//out, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)

	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func getFile(bot *tgbotapi.BotAPI, userID int64, fileID string, userData cache.CacheDataPolling, botstateindex int64) {

	fmt.Printf("Входные данные %v\n\n", userData)

	url, err := bot.GetFileDirectURL(fileID)

	if err != nil {
		zrlog.Fatal().Msg(fmt.Sprintf("bot can't get url's this file: %+v\n", err.Error()))
		log.Printf("FATAL: %v", fmt.Sprintf("bot can't get url's this file: %+v\n", err.Error()))
	} else {

		filename := path.Base(url)

		file_path := fmt.Sprintf("%s/%v_%v_%s", cons.FILE_PATH, userID, botstateindex, filename)

		if botstateindex == botstate.ASK_PHOTO.EnumIndex() {
			userPolling.Set(userID, enumapplic.PHOTO, file_path)
			fmt.Printf("Здесь\n\n")
		}

		if botstateindex == botstate.ASK_FILE.EnumIndex() {
			userPolling.Set(userID, enumapplic.FILE, file_path)
			fmt.Printf("Тут!!!!!!!!\n\n\n")
		}

		err = downloadFile(file_path, url)

		if err != nil {
			zrlog.Fatal().Msg(fmt.Sprintf("bot can't download this file: %+v\n", err.Error()))
			log.Printf("FATAL: %v", fmt.Sprintf("bot can't download this file: %+v\n", err.Error()))

		} else {
			fmt.Printf("Скачан файл!\n\n\n")
		}

		fmt.Printf("Выходные данные %v\n\n", userData)
	}

}

func deleteUserPolling(userID int64, userData cache.CacheDataPolling) {

	userDP := userData.Get(userID)

	fmt.Printf("Сейчас будем удалять опросник:\n\n\n%v\n\n\n\n", userDP)

	//delete user's files and datas in hashmap

	//removing file from the directory
	e := os.Remove(userDP.RequisitionPDF)
	if e != nil {
		zrlog.Fatal().Msg(fmt.Sprintf("Error delete reqisition PDF file: %+v\n", e.Error()))
		log.Printf("ERROR: %v", fmt.Sprintf("Error delete reqisition PDF file: %+v\n", e.Error()))
	}

	e = os.Remove(userDP.Photo)
	if e != nil {
		zrlog.Fatal().Msg(fmt.Sprintf("Error delete file user's foto: %+v\n", e.Error()))
		log.Printf("ERROR: %v", fmt.Sprintf("Error delete file user's foto: %+v\n", e.Error()))
	}

	e = os.Remove(userDP.File)
	if e != nil {
		zrlog.Fatal().Msg(fmt.Sprintf("Error delete file user's (paid check): %+v\n", e.Error()))
		log.Printf("ERROR: %v", fmt.Sprintf("Error delete file user's (paid check): %+v\n", e.Error()))
	}

	userData.Delete(userID)
}
