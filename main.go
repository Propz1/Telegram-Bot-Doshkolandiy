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
			tgbotapi.NewKeyboardButton("Получить диплом"),
		),
	)

	keyboardApplicationStart = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Продолжить"),
			tgbotapi.NewKeyboardButton(botcommand.CANCEL.String()),
		),
	)

	keyboardContinueClosingApplication = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CANCEL.String()),
		),
	)

	keyboardContinueDataPolling1 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CANCEL_APPLICATION.String()),
		),
	)

	keyboardContinueDataPolling2 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Далее"),
			tgbotapi.NewKeyboardButton(botcommand.CANCEL_APPLICATION.String()),
		),
	)

	keyboardConfirm = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CONFIRM.String()),
			tgbotapi.NewKeyboardButton(botcommand.SELECT_CORRECTION.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CANCEL_APPLICATION.String()),
		),
	)

	keyboardConfirmForAdmin = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CONFIRM.String()),
			tgbotapi.NewKeyboardButton(botcommand.CANCEL.String()),
		),
	)

	keyboardAdminMainMenue = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CLOSE_REQUISITION_START.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.SETTINGS.String()),
		),
	)

	contests = map[string]string{
		"Синичка невеличка и ee друзья": "Titmouse",
		"Мама лучший друг":              "Mather",
		"Методическая находка":          "Find",
		"Осень и ee дары":               "Autumn",
	}

	userPolling        = cache.NewCacheDataPolling()
	closingRequisition = cache.NewCacheDataClosingRequisition()
	// msgToUser          string

	botsCommand = [10]string{"CompletedApplication", "SendPublication", "Movements", "MovementsPDF", "/start", "EnterPassword", "Settings", "AppendUser", "ShowUsers", "DeleteUser"}

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

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_PHOTO.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FILE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

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

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

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

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

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

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					return
				}

			}

			messageByteText := bytes.TrimPrefix([]byte(update.Message.Text), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			messageText := string(messageByteText[:])

			switch messageText {

			case "/start":

				err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Здравствуйте, %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.START, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

			case botcommand.CLOSE_REQUISITION_START.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_REQUISITION_NUMBER)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Номер заявки:", nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

				if err != nil {

				}

			case botcommand.COMPLETE_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PROJECT)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите конкурс:", nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.SELECT_PROJECT.String():

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PROJECT {

					if userPolling.Get(update.Message.Chat.ID).Agree {

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP)

					} else {

						err = sentToTelegramm(bot, update.Message.Chat.ID, "Для продолжения необходимо дать согласние на обработку персональных данных. Или нажмите \"Отмена\"", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				}

			case botcommand.CANCEL.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.CANCEL_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case botcommand.START_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP)

				err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.START_APPLICATION, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case "Настройки":

				err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите действие:", nil, cons.StyleTextCommon, botcommand.SETTINGS, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.SETTINGS)

			default:

				stateBot := cacheBotSt.Get(update.Message.Chat.ID)

				switch stateBot {

				case botstate.ASK_PUBLICATION_DATE:

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_DATE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PUBLICATION_LINK)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Укажите ссылку на опубликованную работу:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_LINK, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_PUBLICATION_LINK:

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_LINK, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_REQUISITION_NUMBER:

					_, err := strconv.Atoi(messageText)

					if err != nil {
						zrlog.Info().Msg(fmt.Sprintf("Error convert strconv.Atoi: %+v\n", err.Error()))

						err := sentToTelegramm(bot, update.Message.Chat.ID, "Некорректно введен номер заявки. Введите цифрами:", nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						closingRequisition.Set(update.Message.Chat.ID, enumapplic.REQUISITION_NUMBER, messageText)
						closingRequisition.Set(update.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_DEGREE)

						err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите степень:", nil, cons.StyleTextCommon, botcommand.SELECT_DEGREE, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				case botstate.ASK_FNP:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_AGE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_FNP_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_AGE:

					age, err := strconv.Atoi(messageText)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("Error convert age: %+v\n", err.Error()))
						log.Printf("ERROR: %v", fmt.Sprintf("Error convert age: %+v\n", err.Error()))

						err := sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите, пожалуйста, возраст в правильном формате (цифрой/цифрами):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					} else if age > 120 || age == 0 || age < 0 {

						err := sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Пожалуйста, укажите \"реальный возраст\" (цифрой/цифрами):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				case botstate.ASK_AGE_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAME_INSTITUTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_LOCALITY)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAME_INSTITUTION_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_LOCALITY:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAMING_UNIT)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_LOCALITY_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAMING_UNIT:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_NAMING_UNIT_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_PUBLICATION_TITLE:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP_LEADER)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (нажать \"Далее\" если нет руководителя):", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_FNP_LEADER, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_PUBLICATION_TITLE_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_FNP_LEADER:

					if messageText != botcommand.DOWN.String() {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, "")
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}

				case botstate.ASK_FNP_LEADER_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_EMAIL:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE)

					err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_EMAIL_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				case botstate.ASK_CHECK_DATA:

					if messageText == botcommand.SELECT_CORRECTION.String() {

						cacheBotSt.Set(update.Message.Chat.ID, botstate.SELECT_CORRECTION)

						err = sentToTelegramm(bot, update.Message.Chat.ID, "Выберите пункт который нужно исправить:", nil, cons.StyleTextCommon, botcommand.SELECT_CORRECTION, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else if messageText == botcommand.CONFIRM.String() {

						if !thisIsAdmin(update.Message.Chat.ID) {

							err := sentToTelegramm(bot, update.Message.Chat.ID, "Регистрирую...", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

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

							err = AddRequisition(update.Message.Chat.ID, dbpool, ctx)

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

								fmt.Printf("Not OK")

							} else {

								numReq := userPolling.Get(update.Message.Chat.ID).RequisitionNumber
								path_reqPDF := fmt.Sprintf("./external/files/usersfiles/Заявка_№%v.pdf", numReq)

								userPolling.Set(update.Message.Chat.ID, enumapplic.REQUISITION_PDF, path_reqPDF)

								err = sentToTelegrammPDF(bot, update.Message.Chat.ID, path_reqPDF, "", botcommand.UNDEFINED)

								if err != nil {
									zrlog.Fatal().Msg(fmt.Sprintf("Error sending file pdf to user: %v\n", err))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending file pdf to user: %v\n", err))
									return
								}

								//Email
								t := time.Now()
								formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

								send, err := SentEmail(os.Getenv("ADMIN_EMAIL"), update.Message.Chat.ID, *userPolling, true, fmt.Sprintf("Заявка №%v от %s (%s)", numReq, formattedTime, userPolling.Get(update.Message.Chat.ID).DocumentType), userPolling.Get(update.Message.Chat.ID).Files, "")

								if err != nil {

									zrlog.Fatal().Msg(fmt.Sprintf("Error sending letter to admin's email: %+v\n", err.Error()))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending letter to admin's email: %+v\n", err.Error()))

									fmt.Printf("%v", err)

								}

								if send {

									go deleteUserPolling(update.Message.Chat.ID, *userPolling)

								}

								err = sentToTelegramm(bot, update.Message.Chat.ID, "Поздравляем, Ваша заявка зарегестрирована!", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

								if err != nil {
									zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
									log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
									return
								}

							}

						} else {

							dataForClosing := closingRequisition.Get(update.Message.Chat.ID)

							ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Unable to establish connection to database: %+v\n", err.Error()))
								log.Printf("FATAL: %v", fmt.Sprintf("Unable to establish connection to database: %+v\n", err.Error()))
								os.Exit(1)
								return
							}
							defer dbpool.Close()

							dbpool.Config().MaxConns = 12

							err, userID := GetRequisition(dataForClosing.RequisitionNumber, dataForClosing.TableDB, dataForClosing.Degree, dataForClosing.PublicationDate, dataForClosing.PublicationLink, dbpool, ctx)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Error in GetRequisition(): %+v\n", err.Error()))
							}

							err = FillInPDFForm(userID)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Unable to fill in PDF's form: %+v\n", err.Error()))
							}

							t := time.Now()
							formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

							send, err := SentEmail(userPolling.Get(userID).Email, userID, *userPolling, false, fmt.Sprintf("%s №%v от %s ", userPolling.Get(userID).DocumentType, userPolling.Get(userID).RequisitionNumber, formattedTime), userPolling.Get(userID).Files, "")

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("Error sending letter to admin's email: %+v\n", err.Error()))
							}

							if send {

								err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Заявка №%v закрыта!", dataForClosing.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

								if err != nil {
									zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
									return
								}

								err = UpdateRequisition(dataForClosing.RequisitionNumber, dataForClosing.TableDB, dataForClosing.Degree, dataForClosing.PublicationLink, dataForClosing.PublicationDate, dbpool, ctx)

								if err != nil {
									zrlog.Info().Msg(fmt.Sprintf("Error UpdateRequisition(): %+v\n", err))
								}

								cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

								go deleteUserPolling(userID, *userPolling)
							}

							if !send {

								err = sentToTelegramm(bot, update.Message.Chat.ID, fmt.Sprintf("Не удалось отправить письмо. Заявка №%v не закрыта.", dataForClosing.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

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
					//msgToUser = update.Message.Text
				}

			}
		}

		if update.CallbackQuery != nil {

			callbackQueryData := bytes.TrimPrefix([]byte(update.CallbackQuery.Data), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			callbackQueryText := string(callbackQueryData[:])

			switch callbackQueryText {

			case string(cons.CONTEST_Autumn):

			case string(cons.CONTEST_Titmouse):

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Titmouse.String())

				//Concise description of contest
				description := GetConciseDescription(string(cons.CONTEST_Titmouse))

				err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Mather):

			case string(cons.CONTEST_Find):

			case string(cons.DEGREE1):

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

					closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE1.String())
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case string(cons.DEGREE2):

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE2.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

						err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					}
				}

			case string(cons.DEGREE3):

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE3.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

						err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					}
				}

			case cons.CERTIFICATE.String():

				userID := update.CallbackQuery.Message.Chat.ID

				if !thisIsAdmin(userID) {

					userPolling.Set(userID, enumapplic.DOCUMENT_TYPE, string(cons.CERTIFICATE))
					userPolling.Set(userID, enumapplic.DIPLOMA, "false")
					userPolling.Set(userID, enumapplic.TableDB, cons.TableDB)

					if cacheBotSt.Get(userID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

						cacheBotSt.Set(userID, botstate.ASK_CHECK_DATA)

						err = sentToTelegramm(bot, userID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						cacheBotSt.Set(userID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

						err = sentToTelegramm(bot, userID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}
				}

			case cons.CERTIFICATE_AND_DIPLOMA.String():

				if !thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DOCUMENT_TYPE, string(cons.DIPLOMA))
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DIPLOMA, "true")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

						err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}

					} else {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

						err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
							return
						}
					}
				}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS1:

				cb := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

				if cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS || cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS1)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS2:

				cb := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

				if cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS || cb == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS2)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.CANSEL_CORRECTION.String(): //CallBackQwery "CANSEL_CORRECTION"
				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.FNP.String(): //CallBackQwery "FNP"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v Введите ФИО:", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.START_APPLICATION, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					}

				}

			case enumapplic.AGE.String(): //CallBackQwery "AGE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.NAME_INSTITUTION.String(): //CallBackQwery "NAME_INSTITUTION"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.LOCALITY.String(): //CallBackQwery "LOCALITY"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_LOCALITY_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.NAMING_UNIT.String(): //CallBackQwery "NAMING_UNIT"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAMING_UNIT_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.PUBLICATION_TITLE.String(): //CallBackQwery "PUBLICATION_TITLE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case enumapplic.FNP_LEADER.String(): //CallBackQwery "FNP_LEADER"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_LEADER_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя:", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.EMAIL.String(): //CallBackQwery "EMAIL"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_EMAIL_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.DOCUMENT_TYPE.String(): //CallBackQwery "DOCUMENT_TYPE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String(): //CallBackQwery "PLACE_DELIVERY_OF_DOCUMENTS"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}
				}

			case enumapplic.PHOTO.String(): //CallBackQwery "PHOTO"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case enumapplic.FILE.String(): //CallBackQwery "FILE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FILE_CORRECTION)

					err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						return
					}

				}

			case cons.AGREE.String():

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.AGREE, "")

				err = sentToTelegramm(bot, update.CallbackQuery.Message.Chat.ID, "Согласие на обработку персональных данных получено", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", "", false)

				if err != nil {
					zrlog.Fatal().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					log.Printf("FATAL: %v", fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
					return
				}

			}
		}
	}

}

func sentToTelegramm(bot *tgbotapi.BotAPI, id int64, message string, lenBody map[int]int, styleText string, command botcommand.BotCommand, button string, header string, PDF bool) error {

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
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.CANCEL:

		msg := tgbotapi.NewMessage(id, message, styleText) //enter to main menue

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

		if thisIsAdmin(id) {
			deleteClosingRequisition(id)
		} else {
			deleteUserPolling(id, *userPolling)
		}

	case botcommand.COMPLETE_APPLICATION:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton4 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Autumn.String(), string(cons.CONTEST_Autumn)))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Titmouse.String(), string(cons.CONTEST_Titmouse)))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Mather.String(), string(cons.CONTEST_Mather)))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Find.String(), string(cons.CONTEST_Find)))
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

			msg := tgbotapi.NewMessage(id, message, styleText)

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

	case botcommand.SELECT_DEGREE:

		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.DEGREE1.String(), string(cons.DEGREE1)))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.DEGREE2.String(), string(cons.DEGREE2)))
		rowsButton = append(rowsButton, inlineKeyboardButton2)

		inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.DEGREE3.String(), string(cons.DEGREE3)))
		rowsButton = append(rowsButton, inlineKeyboardButton3)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.GET_PUBLICATION_DATE:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.GET_PUBLICATION_LINK:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
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

	case botcommand.CONTINUE_DATA_POLLING:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardContinueClosingApplication
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
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SELECT_FNP_LEADER:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling2
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SELECT_DOCUMENT_TYPE:

		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.CERTIFICATE), cons.CERTIFICATE.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(string(cons.CERTIFICATE_AND_DIPLOMA), cons.CERTIFICATE_AND_DIPLOMA.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton2)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			zrlog.Fatal().Msg(err.Error())
			return err
		}

	case botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS:

		if !thisIsAdmin(id) {

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.PLACE_DELIVERY_OF_DOCUMENTS1, cons.PLACE_DELIVERY_OF_DOCUMENTS1))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.PLACE_DELIVERY_OF_DOCUMENTS2, cons.PLACE_DELIVERY_OF_DOCUMENTS2))
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

	case botcommand.CHECK_DATA:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if !thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirm
		} else {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

		message = UserDataToStringForTelegramm(id)

		msg = tgbotapi.NewMessage(id, message, cons.StyleTextHTML)

		if !thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirm
		} else {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			zrlog.Panic().Msg(err.Error())
			log.Printf("PANIC: %v", err.Error())
			return err
		}

	case botcommand.SETTINGS:

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
			msg.ReplyMarkup = keyboardAdminMainMenue
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
			msg.ReplyMarkup = keyboardAdminMainMenue
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
		return id == i
	}

	return false
}

func AddRequisition(userID int64, dbpool *pgxpool.Pool, ctx context.Context) error {

	var requisition_number int

	userData := userPolling.Get(userID)

	fmt.Printf("userData.TableDB=%v\n\n", userData.TableDB)

	row, err := dbpool.Query(ctx, fmt.Sprintf("insert into %s (user_id, contest, user_fnp, user_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, document_type, place_delivery_of_document, diploma, start_date, expiration, close_date) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) returning requisition_number", userData.TableDB), userID, userData.Contest, userData.FNP, userData.Age, userData.NameInstitution, userData.Locality, userData.NamingUnit, userData.PublicationTitle, userData.LeaderFNP, userData.Email, userData.DocumentType, userData.PlaceDeliveryDocuments, userData.Diploma, time.Now().UnixNano(), int64(time.Now().Add(172800*time.Second).UnixNano()), 0)

	if err != nil {
		return fmt.Errorf("query to db is failed: %W", err)
	}

	if row.Next() {

		err := row.Scan(&requisition_number)

		if err != nil {
			return fmt.Errorf("scan datas of row is failed %w", err)
		}

		userPolling.Set(userID, enumapplic.REQUISITION_NUMBER, fmt.Sprintf("%v", requisition_number))
	}

	if userData.Diploma {

		var diploma_number int

		row, err := dbpool.Query(ctx, "insert into diplomas (requisition_number) values ($1) returning diploma_number", requisition_number)

		if err != nil {
			return fmt.Errorf("query to db is failed: %W", err)
		}

		if row.Next() {

			err := row.Scan(&diploma_number)

			if err != nil {
				return fmt.Errorf("Scan datas of row is failed %w", err)
			}

			userPolling.Set(userID, enumapplic.DIPLOMA_NUMBER, fmt.Sprintf("%v", diploma_number))
		}

	}

	return row.Err()
}

func GetRequisition(requisition_number int64, tableDB string, degree string, publicationDate, publicationLink string, dbpool *pgxpool.Pool, ctx context.Context) (err error, userID int64) {

	var fnp string
	var age int
	var name_institution string
	var locality string
	var naming_unit string
	var publication_title string
	var leader_fnp string
	var email string
	var contest string
	var document_type string
	var diploma bool
	var diploma_number int64

	row, err := dbpool.Query(ctx, fmt.Sprintf("SELECT user_id, user_fnp, user_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, contest, document_type, diploma, COALESCE(diploma_number, 0) FROM %s LEFT JOIN diplomas ON %s.requisition_number=diplomas.requisition_number WHERE %s.requisition_number = $1", tableDB, tableDB, tableDB), requisition_number)

	if err != nil {
		return fmt.Errorf("Query to db is failed: %W", err), 0
	}

	if row.Next() {

		err = row.Scan(&userID, &fnp, &age, &name_institution, &locality, &naming_unit, &publication_title, &leader_fnp, &email, &contest, &document_type, &diploma, &diploma_number)

		if err != nil {
			return fmt.Errorf("Scan datas of row is failed %w", err), 0
		}

		userPolling.Set(userID, enumapplic.FNP, fnp)
		userPolling.Set(userID, enumapplic.AGE, strconv.Itoa(age))
		userPolling.Set(userID, enumapplic.NAME_INSTITUTION, name_institution)
		userPolling.Set(userID, enumapplic.LOCALITY, locality)
		userPolling.Set(userID, enumapplic.NAMING_UNIT, naming_unit)
		userPolling.Set(userID, enumapplic.PUBLICATION_TITLE, publication_title)
		userPolling.Set(userID, enumapplic.FNP_LEADER, leader_fnp)
		userPolling.Set(userID, enumapplic.EMAIL, email)
		userPolling.Set(userID, enumapplic.CONTEST, contest)
		userPolling.Set(userID, enumapplic.DOCUMENT_TYPE, document_type)
		userPolling.Set(userID, enumapplic.REQUISITION_NUMBER, fmt.Sprintf("%v", requisition_number))
		userPolling.Set(userID, enumapplic.PUBLICATION_LINK, publicationLink)
		userPolling.Set(userID, enumapplic.PUBLICATION_DATE, publicationDate)
		userPolling.Set(userID, enumapplic.DEGREE, degree)
		userPolling.Set(userID, enumapplic.TableDB, cons.CERTIFICATE.String())
		userPolling.Set(userID, enumapplic.DIPLOMA, strconv.FormatBool(diploma))
		if diploma {
			userPolling.Set(userID, enumapplic.DIPLOMA_NUMBER, strconv.Itoa(int(diploma_number)))
		}

	}

	return row.Err(), userID
}

func UpdateRequisition(requisition_number int64, typeDoc string, degree string, publicationLink string, publication_date string, dbpool *pgxpool.Pool, ctx context.Context) (err error) {

	publicationDate := dateStringToUnixNano(publication_date)

	row, err := dbpool.Query(ctx, fmt.Sprintf("UPDATE %s SET reference='%s', publication_date='%v', close_date='%v', degree='%v', email='%s',user_fnp='%s',leader_fnp='%s',user_id='%v' WHERE requisition_number=$1 RETURNING user_id", typeDoc, publicationLink, publicationDate, time.Now().UnixNano(), degree, "", "", "", 0), requisition_number)

	if err != nil {
		return fmt.Errorf("Query \"UPDATE\" to db is failed: %W", err)
	}

	row.Next()

	return row.Err()
}

func FillInPDFForm(userID int64) error {

	var x float64
	var y float64 = 305
	var step float64 = 15
	var nameAndAge string
	var widthText float64
	var centerX float64 = 297.5
	var boilerplatePDFPath string
	var path string

	usersRequisition := userPolling.Get(userID)

	if usersRequisition.LeaderFNP != "" {

		path = "./external/imgs/%s_%s_curator.jpg"

	} else {

		path = "./external/imgs/%s_%s.jpg"

	}

	boilerplatePDFPath = fmt.Sprintf(path, contests[usersRequisition.Contest], cons.CERTIFICATE.String())

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	pdf.Image(boilerplatePDFPath, 0, 0, rect)

	//1. Degree

	pdf.SetXY(241, 211)
	pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
	err = pdf.SetFont("TelegraphLine", "", 24)

	if err != nil {
		zrlog.Info().Msg(err.Error())
	}

	err = pdf.Text(fmt.Sprintf("%v", usersRequisition.Degree))

	if err != nil {
		zrlog.Info().Msg(err.Error())
	}

	//2. Requisition number

	switch {
	case usersRequisition.RequisitionNumber > 10000:
		x = 73
	case usersRequisition.RequisitionNumber > 1000:
		x = 83
	case usersRequisition.RequisitionNumber > 100:
		x = 93
	default:
		x = 103
	}

	pdf.SetXY(x, 245)
	pdf.SetTextColorCMYK(30, 0, 0, 100) //black
	err = pdf.SetFont("TelegraphLine", "", 14)

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.Text(fmt.Sprintf("%v", usersRequisition.RequisitionNumber))

	if err != nil {
		log.Print(err.Error())
	}

	//3. Name and Age

	pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
	err = pdf.SetFont("TelegraphLine", "", 24)

	age_string := strconv.Itoa(usersRequisition.Age)

	var numPrev string
	var numLast string
	var numbers [2]int

	if usersRequisition.Age >= 10 {
		numPrev = age_string[len(age_string)-2 : len(age_string)-1]
	} else {
		numPrev = "0"
	}

	numLast = age_string[len(age_string)-1:]

	numbers[0], _ = strconv.Atoi(numPrev)
	numbers[1], _ = strconv.Atoi(numLast)

	if usersRequisition.Age >= 10 {

		switch numbers[0] {
		case 1:
			age_string = fmt.Sprintf("%s лет", age_string)
		default:

			switch numbers[1] {
			case 1:
				age_string = fmt.Sprintf("%s год", age_string)
			case 2:
				age_string = fmt.Sprintf("%s года", age_string)
			case 3:
				age_string = fmt.Sprintf("%s года", age_string)
			case 4:
				age_string = fmt.Sprintf("%s года", age_string)
			default:
				age_string = fmt.Sprintf("%s лет", age_string)
			}

		}

	} else {

		switch numbers[1] {
		case 1:
			age_string = fmt.Sprintf("%s год", age_string)
		case 2:
			age_string = fmt.Sprintf("%s года", age_string)
		case 3:
			age_string = fmt.Sprintf("%s года", age_string)
		case 4:
			age_string = fmt.Sprintf("%s года", age_string)
		default:
			age_string = fmt.Sprintf("%s лет", age_string)
		}
	}

	nameAndAge = fmt.Sprintf("%s, %s", usersRequisition.FNP, age_string)

	widthText, err = pdf.MeasureTextWidth(nameAndAge)

	x = centerX - widthText/2

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(nameAndAge, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		y = pdf.GetY() + 2*step

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		pdf.SetXY(x, 270)
		err = pdf.Text(nameAndAge)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//4. Name institution

	pdf.SetTextColorCMYK(30, 0, 0, 100) //black
	err = pdf.SetFont("TelegraphLine", "", 18)

	widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution)

	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.NameInstitution, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		x = centerX - widthText/2

		pdf.SetXY(x, 305)
		err = pdf.Text(usersRequisition.NameInstitution)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//5. Locality

	pdf.SetTextColorCMYK(30, 0, 0, 100) //black

	widthText, err = pdf.MeasureTextWidth(usersRequisition.Locality)

	y = pdf.GetY() + 1.5*step

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.Locality, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(usersRequisition.Locality)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//6. Naming unit

	pdf.SetXY(152, 622)
	pdf.SetTextColorCMYK(30, 0, 0, 100) //black

	err = pdf.Text(usersRequisition.NamingUnit)

	if err != nil {
		log.Print(err.Error())
	}

	//6. Publication title

	pdf.SetXY(194, 646)
	pdf.SetTextColorCMYK(30, 0, 0, 100) //black

	err = pdf.Text(usersRequisition.PublicationTitle)

	if err != nil {
		log.Print(err.Error())
	}

	//7. Leader's FNP

	if usersRequisition.LeaderFNP != "" {

		pdf.SetXY(194, 668)

		err = pdf.Text(usersRequisition.LeaderFNP)

		if err != nil {
			log.Print(err.Error())
		}
	}

	//8. Publication date

	pdf.SetXY(355, 718)
	pdf.SetTextColorCMYK(30, 0, 0, 100) //black

	err = pdf.Text(usersRequisition.PublicationDate)

	if err != nil {
		log.Print(err.Error())
	}

	//9. Publication link

	pdf.SetXY(50, 740)
	pdf.SetTextColorCMYK(30, 0, 0, 100) //black

	err = pdf.Text(usersRequisition.PublicationLink)

	if err != nil {
		log.Print(err.Error())
	}

	path = fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.CERTIFICATE), usersRequisition.RequisitionNumber)

	err = pdf.WritePdf(path)

	if err != nil {

		log.Print(err.Error())

		return err
	}

	userPolling.Set(userID, enumapplic.FILE, path)

	err = pdf.Close()

	if !usersRequisition.Diploma {
		return nil
	}

	boilerplatePDFPath = fmt.Sprintf("./external/imgs/%s_%s.jpg", contests[usersRequisition.Contest], cons.DIPLOMA.String())

	pdf = gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err = pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")

	if err != nil {
		log.Print(err.Error())
	}

	pdf.AddPage()
	rect = &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	pdf.Image(boilerplatePDFPath, 0, 0, rect)

	//1. Diploma number

	pdf.SetTextColorCMYK(58, 46, 41, 94) //black
	err = pdf.SetFont("TelegraphLine", "", 14)

	switch {
	case usersRequisition.RequisitionNumber > 10000:
		x = 61
	case usersRequisition.RequisitionNumber > 1000:
		x = 71
	case usersRequisition.RequisitionNumber > 100:
		x = 81
	default:
		x = 91
	}

	pdf.SetXY(x, 262)

	if err != nil {
		log.Print(err.Error())
	}

	err = pdf.Text(fmt.Sprintf("%v", usersRequisition.DiplomaNumber))

	if err != nil {
		log.Print(err.Error())
	}

	//2. Leader's FNP
	pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
	err = pdf.SetFont("TelegraphLine", "", 24)

	widthText, err = pdf.MeasureTextWidth(usersRequisition.LeaderFNP)
	y = 270
	x = centerX - widthText/2

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.LeaderFNP, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		y = pdf.GetY() + 2*step

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		pdf.SetXY(x, y)
		err = pdf.Text(usersRequisition.LeaderFNP)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//3. Name institution

	pdf.SetTextColorCMYK(58, 46, 41, 94) //black
	err = pdf.SetFont("TelegraphLine", "", 18)

	widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution)

	if err != nil {
		log.Print(err.Error())
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.NameInstitution, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		x = centerX - widthText/2

		pdf.SetXY(x, 305)
		err = pdf.Text(usersRequisition.NameInstitution)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//4. Locality

	widthText, err = pdf.MeasureTextWidth(usersRequisition.Locality)

	y = pdf.GetY() + 1.5*step

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.Locality, maxWidthPDF)
		if err != nil {
			log.Print(err.Error())
		}

		for _, t := range arrayText {

			widthText, err = pdf.MeasureTextWidth(t)

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			pdf.Text(t)
			y = y + step
		}

	} else {

		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(usersRequisition.Locality)
		if err != nil {
			log.Print(err.Error())
		}
	}

	//5. FNP
	pdf.SetXY(142, 627)

	err = pdf.SetFont("TelegraphLine", "", 18)

	err = pdf.Text(nameAndAge)
	if err != nil {
		log.Print(err.Error())
	}

	//6. Naming unit

	pdf.SetXY(153, 653)

	err = pdf.Text(usersRequisition.NamingUnit)

	if err != nil {
		log.Print(err.Error())
	}

	//7. Publication title

	pdf.SetXY(195, 674)

	err = pdf.Text(usersRequisition.PublicationTitle)

	if err != nil {
		log.Print(err.Error())
	}

	//8. Requisition number

	pdf.SetXY(139, 697)

	err = pdf.Text(fmt.Sprintf("%v", usersRequisition.RequisitionNumber))

	if err != nil {
		log.Print(err.Error())
	}

	//9. Degree
	pdf.SetXY(226, 717)

	err = pdf.Text(fmt.Sprintf(", %v", usersRequisition.Degree))

	if err != nil {
		log.Print(err.Error())
	}

	//10. Publication date

	pdf.SetXY(445, 736)

	err = pdf.Text(usersRequisition.PublicationDate)

	if err != nil {
		log.Print(err.Error())
	}

	//11. Publication link

	pdf.SetXY(56, 757)

	err = pdf.Text(usersRequisition.PublicationLink)

	if err != nil {
		log.Print(err.Error())
	}

	path = fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.DIPLOMA), usersRequisition.RequisitionNumber)

	err = pdf.WritePdf(path)

	if err != nil {

		log.Print(err.Error())

		return err
	}

	userPolling.Set(userID, enumapplic.FILE, path)

	err = pdf.Close()

	return nil

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

	pdf.SetTextColorCMYK(100, 100, 100, 100)

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	pdf.Image("./external/imgs/RequisitionsBoilerplate.jpg", 0, 0, rect)

	pdf.SetXY(200, 220)
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

	err = pdf.WritePdf(fmt.Sprintf("./external/files/usersfiles/Заявка_№%v.pdf", usersRequisition.RequisitionNumber))

	if err != nil {
		log.Print(err.Error())

		return false, err
	}

	return true, nil
}

func SentEmail(to string, userID int64, userDat cache.CacheDataPolling, toAdmin bool, subject string, files []string, message string) (bool, error) {

	usdat := userDat.Get(userID)

	if toAdmin {
		message = UserDataToString(userID, userDat)
	}

	m := gomail.NewMessage()

	m.SetHeader("From", os.Getenv("BOT_EMAIL"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	if toAdmin {
		m.Embed(usdat.Photo)
	}

	if files != nil && len(files) > 0 {
		for _, path := range files {
			m.Attach(path)
		}
	}

	// Set the email body. You can set plain text or html with text/html
	m.SetBody("text/html", message)

	// Settings for SMTP server
	d := gomail.NewDialer(os.Getenv("SMTP_SERVER"), 465, os.Getenv("BOT_LOGIN_EMAIL"), os.Getenv("BOT_PASSWORD_EMAIL"))

	if err := d.DialAndSend(m); err != nil {
		fmt.Println(err)
		return false, err
	}

	return true, nil
}

func UserDataToString(userID int64, userDat cache.CacheDataPolling) string {

	usdata := userDat.Get(userID)

	var text string

	body := make([]string, 12)

	body = append(body, "<!DOCTYPE html><html lang=\"ru\"><body><dl>")

	body = append(body, "<style type=\"text/css\">BODY {margin: 0; /* Убираем отступы в браузере */}#toplayer {background: #F5FFFA; /* Цвет фона */height: 800px /* Высота слоя */}</style>")

	body = append(body, fmt.Sprintf("<div id=\"toplayer\"><dt><p><b>(%v). %s:</b></p></dt>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.Contest))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", usdata.FNP))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %v</p></dd>", usdata.Age))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.NameInstitution))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", usdata.Locality))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.NamingUnit))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.PublicationTitle))
	text = strings.Join(body, "\n")

	if usdata.LeaderFNP == "" {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v).</b> <s><i><b>%s:</b></i></s></p></dt>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", "-"))
	} else {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.LeaderFNP))
	}
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.Email))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", usdata.DocumentType))
	text = strings.Join(body, "\n")

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

	if !thisIsAdmin(userID) {

		usdata := userPolling.Get(userID)
		body := make([]string, 39)

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
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "___________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()))
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")

	} else {

		data := closingRequisition.Get(userID)
		body := make([]string, 12)

		body = append(body, "___________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.REQUISITION_NUMBER.String()))
		body = append(body, fmt.Sprintf("   %v", data.RequisitionNumber))
		text = strings.Join(body, "\n")

		body = append(body, "___________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.DEGREE.String()))
		body = append(body, fmt.Sprintf("   %s", data.Degree))
		text = strings.Join(body, "\n")

		body = append(body, "___________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PUBLICATION_DATE.String()))
		body = append(body, fmt.Sprintf("   %s", data.PublicationDate))
		text = strings.Join(body, "\n")

		body = append(body, "___________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PUBLICATION_LINK.String()))
		body = append(body, fmt.Sprintf("   %s", data.PublicationLink))
		text = strings.Join(body, "\n")
	}

	return text
}

func GetConciseDescription(contest string) string {

	var text string

	body := make([]string, 14)

	if contest == string(cons.CONTEST_Titmouse) {

		body = append(body, "<b>В заявке потребуется указать следующие данные:\n</b>")
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PHOTO.EnumIndex(), enumapplic.PHOTO.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()))
		body = append(body, "\n")
		body = append(body, fmt.Sprintf("Подробнее с условиями конкурса можно ознакомиться на сайте %s\n", os.Getenv("WEBSITE")))
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
		}

		if botstateindex == botstate.ASK_FILE.EnumIndex() {
			userPolling.Set(userID, enumapplic.FILE, file_path)
		}

		err = downloadFile(file_path, url)

		if err != nil {
			zrlog.Fatal().Msg(fmt.Sprintf("bot can't download this file: %+v\n", err.Error()))
		}

	}

}

func deleteUserPolling(userID int64, userData cache.CacheDataPolling) {

	userDP := userData.Get(userID)

	//delete user's files and datas in hashmap

	//removing file from the directory
	e := os.Remove(userDP.RequisitionPDFpath)
	if e != nil {
		zrlog.Error().Msg(fmt.Sprintf("Error delete reqisition PDF file: %+v\n", e.Error()))
		log.Printf("ERROR: %v", fmt.Sprintf("Error delete reqisition PDF file: %+v\n", e.Error()))
	}

	e = os.Remove(userDP.Photo)
	if e != nil {
		zrlog.Error().Msg(fmt.Sprintf("Error delete file user's foto: %+v\n", e.Error()))
	}

	for _, path := range userDP.Files {

		e = os.Remove(path)
		if e != nil {
			zrlog.Error().Msg(fmt.Sprintf("Error delete file user's (paid check): %+v\n", e.Error()))
		}

	}

	userData.Delete(userID)
}

func deleteClosingRequisition(userID int64) {
	closingRequisition.Delete(userID)
	cacheBotSt.Set(userID, botstate.UNDEFINED)
}

func dateStringToUnixNano(dateString string) int64 {

	var d string
	var m string
	var y string

	sliceDate := strings.Split(dateString, ".")

	for k, v := range sliceDate {

		switch k {
		case 0:
			d = v
		case 1:
			m = v
		case 2:
			y = v
		}
	}

	formatted := fmt.Sprintf("%s-%s-%s", y, m, d)

	date, _ := time.Parse(cons.TimeshortForm, formatted)

	return date.UnixNano()

}
