package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"telegrammBot/cons"
	"telegrammBot/internal/botcommand"
	"telegrammBot/internal/botstate"
	"telegrammBot/internal/cache"
	"telegrammBot/internal/enumapplic"
	"telegrammBot/internal/errs"
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
			tgbotapi.NewKeyboardButton(botcommand.COMPLETE_APPLICATION.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.GET_DIPLOMA.String()),
		),
	)

	keyboardApplicationStart = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CONTINUE.String()),
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
			tgbotapi.NewKeyboardButton(botcommand.FURTHER.String()),
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
			tgbotapi.NewKeyboardButton(botcommand.CANCEL_CLOSE_REQUISITION.String()),
		),
	)

	keyboardConfirmAndSendForAdmin = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.SEND_PDF_FILES.String()),
			tgbotapi.NewKeyboardButton(botcommand.CANCEL_CLOSE_REQUISITION.String()),
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
		"Синичка невеличка и ee друзья":     "Titmouse",
		"Мама лучший друг":                  "Mather",
		"Папа лучший друг":                  "Father",
		"Осень и ee дары":                   "Autumn",
		"Зимушка-зима в гости к нам пришла": "Winter",
		"Снежинки-балеринки":                "Snowflakes",
		"Мой веселый снеговик":              "Snowman",
		"Символ года":                       "Symbol",
		"Сердечки для любимых":              "Heart",
		"Секреты новогодней ёлки":           "Secrets",
		"Покормите птиц зимой":              "BirdsFeeding",
		"Широкая масленица":                 "Shrovetide",
		"В гостях у сказки":                 "Fable",
		"Защитники отечества":               "DefendersFatherland",
	}

	tempUsersIDCache   = cache.NewTempUsersIDCache()
	userPolling        = cache.NewCacheDataPolling()
	closingRequisition = cache.NewCacheDataClosingRequisition()
	cellOption_Caption = gopdf.CellOption{Align: 16}
	cellOption_Default = gopdf.CellOption{Align: 8}

	wg sync.WaitGroup

	maxWidthPDF float64 = 507.0

	cacheBotSt cache.CacheBotSt
)

func main() {

	logFile, err := os.OpenFile("./temp/info.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("the file info.log doesn't open: %v", err))
		os.Exit(1)
	}
	defer logFile.Close()

	zrlog.Logger = zerolog.New(logFile).With().Timestamp().Logger()

	err = godotenv.Load("app.env")
	if err != nil {
		zrlog.Fatal().Msg("Error loading .env file: ")
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		os.Exit(1)
	}

	bot.Debug = true

	zrlog.Info().Msg(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))

	webHookInfo := tgbotapi.NewWebhookWithCert(fmt.Sprintf("https://%s:%s/%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT"), bot.Token), cons.CERT_PAHT)

	_, err = bot.SetWebhook(webHookInfo)
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		os.Exit(1)
	}

	infoWebhook, err := bot.GetWebhookInfo()

	if err != nil {
		zrlog.Error().Msg(err.Error())
	}

	if infoWebhook.LastErrorDate != 0 {
		zrlog.Error().Msg(fmt.Sprintf("Telegram callback failed: %s", infoWebhook.LastErrorMessage))
	}

	updates := bot.ListenForWebhook("/" + bot.Token)

	zrlog.Info().Msg(fmt.Sprintf("Starting API server on %s:%s\n", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT")))

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

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("update.Message.Photo != nil, error sending to user: %+v\n", err))
					}

				} else if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_FILE || cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_FILE_CORRECTION {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_FILE.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("update.Message.Photo != nil, error sending to user: %+v\n", err))
					}

				}

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO_CORRECTION {

					ph := *update.Message.Photo

					max_quality := len(ph) - 1

					go getFile(bot, update.Message.Chat.ID, ph[max_quality].FileID, *userPolling, botstate.ASK_PHOTO.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO_CORRECTION, error sending to user: %+v\n", err))
					}

				}

			}

			if update.Message.Document != nil {

				ph := *update.Message.Document

				go getFile(bot, update.Message.Chat.ID, ph.FileID, *userPolling, botstate.ASK_FILE.EnumIndex())

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("update.Message.Document != nil, error sending to user: %+v\n", err))
				}

			}

			messageByteText := bytes.TrimPrefix([]byte(update.Message.Text), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			messageText := string(messageByteText[:])

			switch messageText {

			case botcommand.START.String():

				err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Здравствуйте, %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.START, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.START.String(), error sending to user: %+v\n", err))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

			case botcommand.GET_DIPLOMA.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.GET_DIPLOMA)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Номер заявки:", nil, cons.StyleTextCommon, botcommand.GET_DIPLOMA, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.GET_DIPLOMA.String(), error sending to user: %+v\n", err))
				}

			case botcommand.CLOSE_REQUISITION_START.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_REQUISITION_NUMBER)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Номер заявки:", nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CLOSE_REQUISITION_START.String(), error sending to user: %+v\n", err))
				}

			case botcommand.COMPLETE_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PROJECT)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите конкурс:", nil, cons.StyleTextCommon, botcommand.COMPLETE_APPLICATION, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.COMPLETE_APPLICATION.String(), error sending to user: %+v\n", err))
				}

			case botcommand.SELECT_PROJECT.String():

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PROJECT {

					if userPolling.Get(update.Message.Chat.ID).Agree {

						err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО участника или группу участников (например, \"страшая группа №7\" или \"старшая группа \"Карамельки\"):", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botcommand.SELECT_PROJECT.String(), error sending to user: %+v\n", err))
						}

						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP)

					} else {

						err = sentToTelegram(bot, update.Message.Chat.ID, "Для продолжения необходимо дать согласние на обработку персональных данных. Или нажмите \"Отмена\"", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botcommand.SELECT_PROJECT.String(), error sending to user: %+v\n", err))
						}
					}

				}

			case botcommand.CANCEL.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				if thisIsAdmin(update.Message.Chat.ID) {
					go deleteClosingRequisition(update.Message.Chat.ID)
				} else {
					go deleteUserPolling(update.Message.Chat.ID, *userPolling)
					go checkUsersIDCache(update.Message.Chat.ID, bot)
				}

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CANCEL.String(), error sending to user: %+v\n", err))
				}

			case botcommand.CANCEL_APPLICATION.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.START)

				if thisIsAdmin(update.Message.Chat.ID) {
					go deleteClosingRequisition(update.Message.Chat.ID)
				} else {
					go deleteUserPolling(update.Message.Chat.ID, *userPolling)
					go checkUsersIDCache(update.Message.Chat.ID, bot)
				}

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CANCEL_APPLICATION.String(), error sending to user: %+v\n", err.Error()))
				}

			case botcommand.SETTINGS.String():

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите действие:", nil, cons.StyleTextCommon, botcommand.SETTINGS, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.SETTINGS.String(), error sending to user: %+v\n", err))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.SETTINGS)

			default:

				stateBot := cacheBotSt.Get(update.Message.Chat.ID)

				switch stateBot {

				case botstate.GET_DIPLOMA:

					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))

					if err != nil {
						zrlog.Fatal().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, unable to establish connection to database: %+v\n", err.Error()))
						os.Exit(1)
					}
					defer dbpool.Close()

					dbpool.Config().MaxConns = 12

					requisitionNumber, err := strconv.Atoi(messageText)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, unable to convert string to int (strconv.Atoi): %+v\n", err.Error()))

						err := sentToTelegram(bot, update.Message.Chat.ID, "Некорректно введен номер заявки. Введите цифрами:", nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, error sending to user: %+v\n", err))
						}

					} else {

						err, userID, sent := GetRequisitionForUser(update.Message.Chat.ID, int64(requisitionNumber), dbpool, ctx)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, GetRequisitionForUser(): %+v\n", err))
						}

						switch {

						case sent:

							err := sentToTelegram(bot, update.Message.Chat.ID, "Данная заявка закрыта, диплом/грамота Вам уже были отправлены.", nil, cons.StyleTextCommon, botcommand.ACCESS_DENIED, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, error sending to user: %+v\n", err))
							}

						case update.Message.Chat.ID != userID:

							err := sentToTelegram(bot, update.Message.Chat.ID, "Вы не регистрировали эту заявку.", nil, cons.StyleTextCommon, botcommand.ACCESS_DENIED, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, error sending to user: %+v\n", err))
							}

						case strings.TrimSpace(userPolling.Get(update.Message.Chat.ID).PublicationLink) == "":

							err := sentToTelegram(bot, update.Message.Chat.ID, "Ваша заявка находится в работе.", nil, cons.StyleTextCommon, botcommand.ACCESS_DENIED, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, error sending to user: %+v\n", err))
							}

						default:

							wg.Add(2)
							go FillInCertificatesPDFForms(&wg, userID, *userPolling)
							go FillInDiplomasPDFForms(&wg, userID, *userPolling)
							wg.Wait()

							temp := ""
							for _, path := range userPolling.Get(userID).Files {

								//When some files are the same
								if temp != "" && temp == path {
									continue
								}

								err = sentToTelegramPDF(bot, update.Message.Chat.ID, path, "", botcommand.UNDEFINED)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, error sending file pdf to user: %v\n", err))
								}

								temp = path

							}

							err = UpdateRequisition(false, false, userPolling.Get(userID).RequisitionNumber, userPolling.Get(userID).TableDB, 0, "", "", dbpool, ctx)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.GET_DIPLOMA, UpdateRequisition(): %+v\n", err))
							}

							cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

							go deleteUserPolling(userID, *userPolling)
							go checkUsersIDCache(userID, bot)

						}

					}

				case botstate.ASK_PUBLICATION_DATE:

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_DATE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PUBLICATION_LINK)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Укажите ссылку на опубликованную работу:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_LINK, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_PUBLICATION_DATE, error sending to user: %+v\n", err))
					}

				case botstate.ASK_PUBLICATION_LINK:

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_LINK, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_PUBLICATION_LINK, error sending to user: %+v\n", err))
					}

				case botstate.ASK_REQUISITION_NUMBER:

					_, err := strconv.Atoi(messageText)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_REQUISITION_NUMBER, error convert strconv.Atoi: %+v\n", err))

						err := sentToTelegram(bot, update.Message.Chat.ID, "Некорректно введен номер заявки. Введите цифрами:", nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_REQUISITION_NUMBER, error sending to user: %+v\n", err))
						}

					} else {

						closingRequisition.Set(update.Message.Chat.ID, enumapplic.REQUISITION_NUMBER, messageText)
						closingRequisition.Set(update.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_DEGREE)

						err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите степень:", nil, cons.StyleTextCommon, botcommand.SELECT_DEGREE, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_REQUISITION_NUMBER, error sending to user: %+v\n", err))
						}
					}

				case botstate.ASK_FNP:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FORMAT_CHOICE)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите как вы хотите ввести возраст участника/участников?", nil, cons.StyleTextCommon, botcommand.FORMAT_CHOICE, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_FNP, error sending to user: %+v\n", err))
					}

				case botstate.ASK_FNP_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText)

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_FNP_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_AGE:

					if userPolling.Get(update.Message.Chat.ID).Group {

						userPolling.Set(update.Message.Chat.ID, enumapplic.GROUP_AGE, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION)

						err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE, error sending to user: %+v\n", err))
						}

					} else {

						age, err := strconv.Atoi(messageText)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE, error convert age: %+v\n", err.Error()))

							err := sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите, пожалуйста, возраст в правильном формате (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE, error sending to user: %+v\n", err))
							}
						} else if age > 120 || age < 0 {

							err := sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Пожалуйста, укажите \"реальный возраст\" (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE, error sending to user: %+v\n", err))
							}

						} else {

							userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
							cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION)

							err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE, error sending to user: %+v\n", err))
							}
						}
					}

				case botstate.ASK_AGE_CORRECTION:

					if userPolling.Get(update.Message.Chat.ID).Group {
						userPolling.Set(update.Message.Chat.ID, enumapplic.GROUP_AGE, messageText)
						userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, "0")
					} else {
						userPolling.Set(update.Message.Chat.ID, enumapplic.AGE, messageText)
					}

					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_AGE_CORRECTION, error sending to user: %+v\n", err.Error()))
					}

				case botstate.ASK_NAME_INSTITUTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_LOCALITY)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_NAME_INSTITUTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_NAME_INSTITUTION_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAME_INSTITUTION, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_NAME_INSTITUTION_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_LOCALITY:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_NAMING_UNIT)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_LOCALITY, error sending to user: %+v\n", err))
					}

				case botstate.ASK_LOCALITY_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.LOCALITY, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_LOCALITY_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_NAMING_UNIT:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_NAMING_UNIT, error sending to user: %+v\n", err))
					}

				case botstate.ASK_NAMING_UNIT_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NAMING_UNIT, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_NAMING_UNIT_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_PUBLICATION_TITLE:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_FNP_LEADER)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (через запятую, если двое) или нажмите \"Далее\" если нет руководителя:", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_FNP_LEADER, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_PUBLICATION_TITLE, error sending to user: %+v\n", err))
					}

				case botstate.ASK_PUBLICATION_TITLE_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PUBLICATION_TITLE, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_PUBLICATION_TITLE_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_FNP_LEADER:

					if messageText != botcommand.DOWN.String() {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_FNP_LEADER, error sending to user: %+v\n", err))
						}

					} else {

						userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, "")
						cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_EMAIL)

						err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_FNP_LEADER, error sending to user: %+v\n", err))
						}
					}

				case botstate.ASK_FNP_LEADER_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP_LEADER, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_FNP_LEADER_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_EMAIL:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, strings.TrimSpace(messageText))
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_EMAIL, error sending to user: %+v\n", err))
					}

				case botstate.ASK_EMAIL_CORRECTION:

					userPolling.Set(update.Message.Chat.ID, enumapplic.EMAIL, strings.TrimSpace(messageText))
					cacheBotSt.Set(update.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_EMAIL_CORRECTION, error sending to user: %+v\n", err))
					}

				case botstate.ASK_CHECK_DATA:

					if messageText == botcommand.SELECT_CORRECTION.String() {

						cacheBotSt.Set(update.Message.Chat.ID, botstate.SELECT_CORRECTION)

						err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите пункт который нужно исправить:", nil, cons.StyleTextCommon, botcommand.SELECT_CORRECTION, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA: %+v\n", err))
						}

					} else if messageText == botcommand.CONFIRM.String() {

						if !thisIsAdmin(update.Message.Chat.ID) {

							err := sentToTelegram(bot, update.Message.Chat.ID, "Регистрирую...", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, sending for user: %+v\n", err))
							}

							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, unable to establish connection to database for users: %+v\n", err.Error()))
								os.Exit(1)
							}
							defer dbpool.Close()

							dbpool.Config().MaxConns = 7

							err = AddRequisition(update.Message.Chat.ID, dbpool, ctx)

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, error append requisition to db for user: %+v\n", err.Error()))
								os.Exit(1)
							}

							ok, err := ConvertRequisitionToPDF(update.Message.Chat.ID)

							if err != nil || !ok {

								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, error converting requisition into PDF for user: %+v\n", err.Error()))

							} else {

								numReq := userPolling.Get(update.Message.Chat.ID).RequisitionNumber
								path_reqPDF := fmt.Sprintf("./external/files/usersfiles/Заявка_№%v.pdf", numReq)

								userPolling.Set(update.Message.Chat.ID, enumapplic.REQUISITION_PDF, path_reqPDF)

								err = sentToTelegramPDF(bot, update.Message.Chat.ID, path_reqPDF, "", botcommand.UNDEFINED)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, error sending file pdf to user: %v\n", err))
								}

								//Email
								t := time.Now()
								formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

								send, err := SentEmail(os.Getenv("ADMIN_EMAIL"), update.Message.Chat.ID, *userPolling, true, fmt.Sprintf("Заявка №%v от %s (%s)", numReq, formattedTime, userPolling.Get(update.Message.Chat.ID).DocumentType), userPolling.Get(update.Message.Chat.ID).Files, "")

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, error sending letter to admin's email: %+v\n", err.Error()))
								}

								if send {

									cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)
									go deleteUserPolling(update.Message.Chat.ID, *userPolling)
									go checkUsersIDCache(update.Message.Chat.ID, bot)

								}

								err = sentToTelegram(bot, update.Message.Chat.ID, "Поздравляем, Ваша заявка зарегестрирована! Благодарим Вас за участие, ваша заявка будет обработана в течение трех дней.", nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								}

							}

						}

						if thisIsAdmin(update.Message.Chat.ID) {

							ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
							dbpool.Config().MaxConns = 12

							if err != nil {
								zrlog.Fatal().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, unable to establish connection to database for admin: %+v\n", err.Error()))
								os.Exit(1)
							}
							defer dbpool.Close()

							err, userID := GetRequisitionForAdmin(*userPolling, closingRequisition.Get(update.Message.Chat.ID).RequisitionNumber, closingRequisition.Get(update.Message.Chat.ID).TableDB, closingRequisition.Get(update.Message.Chat.ID).Degree, closingRequisition.Get(update.Message.Chat.ID).PublicationDate, closingRequisition.Get(update.Message.Chat.ID).PublicationLink, dbpool, ctx)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, GetRequisitionForAdmin(): %+v\n", err.Error()))

								err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Ошибка закрытия заявки!\n%s", err.Error()), nil, cons.StyleTextCommon, botcommand.CHECK_DATA_PAUSE, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, sending for admin: %+v\n", err.Error()))
								}

							} else {

								closingRequisition.Set(update.Message.Chat.ID, enumapplic.USER_ID, strconv.Itoa(int(userID)))

								wg.Add(2)
								go FillInCertificatesPDFForms(&wg, userID, *userPolling)
								go FillInDiplomasPDFForms(&wg, userID, *userPolling)
								wg.Wait()

								//Send to admin for check
								temp := ""
								for _, path := range userPolling.Get(userID).Files {

									//When some files are the same
									if temp != "" && temp == path {
										continue
									}

									err = sentToTelegramPDF(bot, update.Message.Chat.ID, path, "", botcommand.UNDEFINED)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, send to admin for check, error sending file pdf to admin: %v\n", err))
									}

									temp = path
								}

								err = sentToTelegram(bot, update.Message.Chat.ID, "Подтвердить или отменить закрытие?", nil, cons.StyleTextCommon, botcommand.CHECK_PDF_FILES, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.ASK_CHECK_DATA, sending for admin: %+v\n", err.Error()))
								}
							}
						}

					} else if messageText == botcommand.SEND_PDF_FILES.String() && thisIsAdmin(update.Message.Chat.ID) {

						ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()

						dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
						dbpool.Config().MaxConns = 12

						if err != nil {
							zrlog.Fatal().Msg(fmt.Sprintf(" else if messageText == botcommand.SEND_PDF_FILES.String() && thisIsAdmin(update.Message.Chat.ID), Unable to establish connection to database: %+v\n", err.Error()))
							os.Exit(1)
						}
						defer dbpool.Close()

						userID := closingRequisition.Get(update.Message.Chat.ID).UserID

						switch userPolling.Get(userID).PlaceDeliveryDocuments {

						case cons.PLACE_DELIVERY_OF_DOCUMENTS1: //Email

							t := time.Now()
							formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

							sent, err := SentEmail(userPolling.Get(userID).Email, userID, *userPolling, false, fmt.Sprintf("%s №%v от %s ", userPolling.Get(userID).DocumentType, userPolling.Get(userID).RequisitionNumber, formattedTime), userPolling.Get(userID).Files, "")

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS1:, error sending letter to admin's email: %+v\n", err.Error()))
							}

							if sent {

								err = UpdateRequisition(true, true, userPolling.Get(userID).RequisitionNumber, userPolling.Get(userID).TableDB, userPolling.Get(userID).Degree, userPolling.Get(userID).PublicationLink, userPolling.Get(userID).PublicationDate, dbpool, ctx)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS1:, UpdateRequisition() for admin: %v\n", err))

									err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Ошибка! Заявка №%v НЕ закрыта!", userPolling.Get(userID).RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS1: %v\n", err))
									}

									cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

									//thisIsAdmin == true, therefore
									// When RequisitionNumber == 0, most likely or the user is working on a new application, or the map "userPolling" is empty therefore, we do not clean in this case
									if userPolling.Get(userID).RequisitionNumber != 0 {
										go deleteUserPolling(userID, *userPolling)
										go checkUsersIDCache(userID, bot)
									}

								} else {

									err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Заявка №%v закрыта!", userPolling.Get(userID).RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS1: %+v\n", err.Error()))
									}

									cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

									// When RequisitionNumber == 0, most likely or the user is working on a new application, or the map "userPolling" is empty therefore, we do not clean in this case
									if userPolling.Get(closingRequisition.Get(update.Message.Chat.ID).UserID).RequisitionNumber != 0 {
										go deleteUserPolling(closingRequisition.Get(update.Message.Chat.ID).UserID, *userPolling)
										go checkUsersIDCache(closingRequisition.Get(update.Message.Chat.ID).UserID, bot)
									}

									go deleteClosingRequisition(update.Message.Chat.ID)
								}

							}

							if !sent {

								err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Не удалось отправить письмо. Заявка №%v не закрыта.", userPolling.Get(userID).RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								}
							}

						case cons.PLACE_DELIVERY_OF_DOCUMENTS2: //Telegram

							err = UpdateRequisition(true, false, userPolling.Get(userID).RequisitionNumber, userPolling.Get(userID).TableDB, userPolling.Get(userID).Degree, userPolling.Get(userID).PublicationLink, userPolling.Get(userID).PublicationDate, dbpool, ctx)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS2, UpdateRequisition() for admin: %+v\n", err))
							}

							err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Заявка №%v закрыта!", userPolling.Get(userID).RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RECORD_TO_DB, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PLACE_DELIVERY_OF_DOCUMENTS2: %+v\n", err.Error()))
							}

							cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

							if !thisIsAdmin(update.Message.Chat.ID) {
								go deleteUserPolling(update.Message.Chat.ID, *userPolling)
								go checkUsersIDCache(update.Message.Chat.ID, bot)
							}

							if thisIsAdmin(update.Message.Chat.ID) {

								// When RequisitionNumber == 0, most likely or the user is working on a new application, or the map "userPolling" is empty therefore, we do not clean in this case
								if userPolling.Get(closingRequisition.Get(update.Message.Chat.ID).UserID).RequisitionNumber != 0 {
									go deleteUserPolling(closingRequisition.Get(update.Message.Chat.ID).UserID, *userPolling)
									go checkUsersIDCache(closingRequisition.Get(update.Message.Chat.ID).UserID, bot)
								}

								go deleteClosingRequisition(update.Message.Chat.ID)
							}

						}

					} else if messageText == botcommand.CANCEL_APPLICATION.String() {

						cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

						if thisIsAdmin(update.Message.Chat.ID) {

							// When RequisitionNumber == 0, most likely or the user is working on a new application, or the map "userPolling" is empty therefore, we do not clean in this case
							if userPolling.Get(closingRequisition.Get(update.Message.Chat.ID).UserID).RequisitionNumber != 0 {
								go deleteUserPolling(closingRequisition.Get(update.Message.Chat.ID).UserID, *userPolling)
								go checkUsersIDCache(closingRequisition.Get(update.Message.Chat.ID).UserID, bot)
							}

							go deleteClosingRequisition(update.Message.Chat.ID)
						}

						if !thisIsAdmin(update.Message.Chat.ID) {
							go deleteUserPolling(update.Message.Chat.ID, *userPolling)
							go checkUsersIDCache(update.Message.Chat.ID, bot)
						}

					} else if messageText == botcommand.CANCEL_CLOSE_REQUISITION.String() && thisIsAdmin(update.Message.Chat.ID) { //excess condition

						err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.CANCEL, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						}

						cacheBotSt.Set(update.Message.Chat.ID, botstate.UNDEFINED)

						// When RequisitionNumber == 0, most likely or the user is working on a new application, or the map "userPolling" is empty therefore, we do not clean in this case
						if userPolling.Get(closingRequisition.Get(update.Message.Chat.ID).UserID).RequisitionNumber != 0 {
							go deleteUserPolling(closingRequisition.Get(update.Message.Chat.ID).UserID, *userPolling)
							go checkUsersIDCache(closingRequisition.Get(update.Message.Chat.ID).UserID, bot)
						}

						go deleteClosingRequisition(update.Message.Chat.ID)

					}

				case botstate.UNDEFINED:
					//msgToUser = update.Message.Text
				}

			}
		}

		if update.CallbackQuery != nil {

			callbackQueryData := bytes.TrimPrefix([]byte(update.CallbackQuery.Data), []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"
			callbackQueryText := string(callbackQueryData[:])

			var description string

			switch callbackQueryText {

			case cons.FORMAT_CHOICE_SINGL.String(): //CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_FORMAT_CHOICE {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.NOT_GROUP, "")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.GROUP_AGE, "")

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.FORMAT_CHOICE_SINGL.String(): %+v\n", err.Error()))
					}

				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_FORMAT_CHOICE_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.NOT_GROUP, "")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.GROUP_AGE, "")

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника (цифрой):", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.FORMAT_CHOICE_SINGL.String(): %+v\n", err.Error()))
					}

				}

			case cons.FORMAT_CHOICE_GROUP.String(): //CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_FORMAT_CHOICE {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.GROUP, "")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.AGE, "0")

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст в произвольном формате:", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.FORMAT_CHOICE_GROUP.String(): %+v\n", err.Error()))
					}

				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_FORMAT_CHOICE_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.GROUP, "")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.AGE, "0")

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_AGE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст в произвольном формате:", enumapplic.AGE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.FORMAT_CHOICE_GROUP.String(): %+v\n", err.Error()))
					}

				}

			case string(cons.CONTEST_Titmouse): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Titmouse.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Titmouse))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, 	case string(cons.CONTEST_Titmouse): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_DefendersFatherland): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_DefendersFatherland.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_DefendersFatherland))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_DefendersFatherland): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Mather): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Mather.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Mather))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Mather): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Father): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Father.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Father))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Father): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Autumn): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Autumn.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Autumn))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Autumn): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Winter): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Winter.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Winter))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Winter): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Snowflakes): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Snowflakes.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Snowflakes))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Snowflakes): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Snowman): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Snowman.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Snowman))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Snowman): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Symbol): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Symbol.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Symbol))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Symbol): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Heart): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Heart.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Heart))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Heart): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Secrets): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Secrets.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Secrets))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Secrets): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_BirdsFeeding): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_BirdsFeeding.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_BirdsFeeding))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_BirdsFeeding): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Shrovetide): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Shrovetide.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Shrovetide))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Shrovetide): %+v\n", err.Error()))
				}

			case string(cons.CONTEST_Fable): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.CONTEST, cons.CONTEST_Fable.String())

				//Concise description of contest
				description = GetConciseDescription(string(cons.CONTEST_Fable))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SELECT_PROJECT, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.CONTEST_Fable): %+v\n", err.Error()))
				}

			case string(cons.DEGREE1): //CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

					closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE1.String())
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.DEGREE1): %+v\n", err.Error()))
					}

				}

			case string(cons.DEGREE2): //CallBackQwery

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE2.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.DEGREE2): %+v\n", err.Error()))
						}

					}
				}

			case string(cons.DEGREE3): //CallBackQwery

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DEGREE {

						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DEGREE, cons.DEGREE3.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_DATE)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GET_PUBLICATION_DATE, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.DEGREE3): %+v\n", err.Error()))
						}

					}
				}

			case cons.CERTIFICATE.String(): //CallBackQwery

				if !thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DOCUMENT_TYPE, string(cons.CERTIFICATE))
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DIPLOMA, "false")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CERTIFICATE.String(): %+v\n", err.Error()))
						}

					} else {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CERTIFICATE.String(): %+v\n", err.Error()))
						}
					}
				}

			case cons.CERTIFICATE_AND_DIPLOMA.String(): //CallBackQwery

				if !thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DOCUMENT_TYPE, string(cons.DIPLOMA))
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DIPLOMA, "true")
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_DOCUMENT_TYPE_CORRECTION {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CERTIFICATE_AND_DIPLOMA.String(): %+v\n", err.Error()))
						}

					} else {

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CERTIFICATE_AND_DIPLOMA.String(): %+v\n", err.Error()))
						}
					}
				}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS1: //CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS1)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.PLACE_DELIVERY_OF_DOCUMENTS1: %+v\n", err.Error()))
					}

				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS1)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.PLACE_DELIVERY_OF_DOCUMENTS1: %+v\n", err.Error()))
					}
				}

			case cons.PLACE_DELIVERY_OF_DOCUMENTS2: //CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS2)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.PLACE_DELIVERY_OF_DOCUMENTS2: %+v\n", err.Error()))
					}

				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION {

					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, cons.PLACE_DELIVERY_OF_DOCUMENTS2)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.PLACE_DELIVERY_OF_DOCUMENTS2: %+v\n", err.Error()))
					}

				}

			case enumapplic.CANCEL_CORRECTION.String(): //CallBackQwery "CANCEL_CORRECTION"
				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_CHECK_DATA)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CHECK_DATA, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.CANCEL_CORRECTION.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.FNP.String(): //CallBackQwery "FNP"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО участника или группу участников (например, \"страшая группа №7\" или \"старшая группа \"Карамельки\"):", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FNP.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.AGE.String(): //CallBackQwery "AGE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FORMAT_CHOICE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Выберите, как вы хотите ввести возраст участника/группы участников?", nil, cons.StyleTextCommon, botcommand.FORMAT_CHOICE, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.AGE.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.NAME_INSTITUTION.String(): //CallBackQwery "NAME_INSTITUTION"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAME_INSTITUTION_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NAME_INSTITUTION.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.NAME_INSTITUTION.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.LOCALITY.String(): //CallBackQwery "LOCALITY"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_LOCALITY_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.LOCALITY.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.LOCALITY.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.NAMING_UNIT.String(): //CallBackQwery "NAMING_UNIT"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_NAMING_UNIT_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NAMING_UNIT.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.NAMING_UNIT.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.PUBLICATION_TITLE.String(): //CallBackQwery "PUBLICATION_TITLE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PUBLICATION_TITLE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PUBLICATION_TITLE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.PUBLICATION_TITLE.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.FNP_LEADER.String(): //CallBackQwery "FNP_LEADER"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FNP_LEADER_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (через запятую, если два):", enumapplic.FNP_LEADER.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FNP_LEADER.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.EMAIL.String(): //CallBackQwery "EMAIL"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_EMAIL_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты:", enumapplic.EMAIL.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.EMAIL.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.DOCUMENT_TYPE.String(): //CallBackQwery "DOCUMENT_TYPE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_DOCUMENT_TYPE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DOCUMENT_TYPE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_DOCUMENT_TYPE, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.DOCUMENT_TYPE.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String(): //CallBackQwery "PLACE_DELIVERY_OF_DOCUMENTS"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String: %+v\n", err.Error()))
					}
				}

			case enumapplic.PHOTO.String(): //CallBackQwery "PHOTO"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_PHOTO_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фото Вашей работы:", enumapplic.PHOTO.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.PHOTO.String(): %+v\n", err.Error()))
					}

				}

			case enumapplic.FILE.String(): //CallBackQwery "FILE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SELECT_CORRECTION {

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.ASK_FILE_CORRECTION)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.FILE.EnumIndex()), nil, cons.StyleTextCommon, botcommand.CONTINUE_DATA_POLLING, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FILE.String(): %+v\n", err.Error()))
					}

				}

			case cons.AGREE.String():

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.AGREE, "")

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Согласие на обработку персональных данных получено", nil, cons.StyleTextCommon, botcommand.WAITING_FOR_ACCEPTANCE, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

			}
		}
	}

}

func sentToTelegram(bot *tgbotapi.BotAPI, id int64, message string, lenBody map[int]int, styleText string, command botcommand.BotCommand, button string, header string, PDF bool) error {

	switch command {

	case botcommand.FORMAT_CHOICE:
		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.FORMAT_CHOICE_SINGL), cons.FORMAT_CHOICE_SINGL.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(string(cons.FORMAT_CHOICE_GROUP), cons.FORMAT_CHOICE_GROUP.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton2)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.FORMAT_CHOICE: %w", err)
		}

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
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_CORRECTIONT: %w", err)
			}

			message = "или"
			msg = tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = keyboardContinueDataPolling1

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_CORRECTIONT: %w", err)
			}

			message = "нажмите"
			msg = tgbotapi.NewMessage(id, message, styleText)

			rowsButton = nil
			inlineKeyboardButton13 = append(inlineKeyboardButton13, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s", enumapplic.CANCEL_CORRECTION.String()), enumapplic.CANCEL_CORRECTION.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton13)
			inlineKeyboardMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_CORRECTIONT: %w", err)
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
			return fmt.Errorf("sentToTelegram(), botcommand.START: %w", err)
		}

	case botcommand.ACCESS_DENIED:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardMainMenue

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.ACCESS_DENIED: %w", err)
		}

		deleteUserPolling(id, *userPolling)
		go checkUsersIDCache(id, bot)

	case botcommand.CANCEL:

		msg := tgbotapi.NewMessage(id, message, styleText) //enter to main menue

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CANCEL: %w", err)
		}

		if thisIsAdmin(id) {
			deleteClosingRequisition(id)
		} else {
			deleteUserPolling(id, *userPolling)
			go checkUsersIDCache(id, bot)
		}

	case botcommand.COMPLETE_APPLICATION:

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
			inlineKeyboardButton14 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Titmouse.String(), string(cons.CONTEST_Titmouse)))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Mather.String(), string(cons.CONTEST_Mather)))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Father.String(), string(cons.CONTEST_Father)))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Autumn.String(), string(cons.CONTEST_Autumn)))
			rowsButton = append(rowsButton, inlineKeyboardButton4)

			inlineKeyboardButton5 = append(inlineKeyboardButton5, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Winter.String(), string(cons.CONTEST_Winter)))
			rowsButton = append(rowsButton, inlineKeyboardButton5)

			inlineKeyboardButton6 = append(inlineKeyboardButton6, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Snowflakes.String(), string(cons.CONTEST_Snowflakes)))
			rowsButton = append(rowsButton, inlineKeyboardButton6)

			inlineKeyboardButton7 = append(inlineKeyboardButton7, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Snowman.String(), string(cons.CONTEST_Snowman)))
			rowsButton = append(rowsButton, inlineKeyboardButton7)

			inlineKeyboardButton8 = append(inlineKeyboardButton8, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Symbol.String(), string(cons.CONTEST_Symbol)))
			rowsButton = append(rowsButton, inlineKeyboardButton8)

			inlineKeyboardButton9 = append(inlineKeyboardButton9, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Heart.String(), string(cons.CONTEST_Heart)))
			rowsButton = append(rowsButton, inlineKeyboardButton9)

			inlineKeyboardButton10 = append(inlineKeyboardButton10, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Secrets.String(), string(cons.CONTEST_Secrets)))
			rowsButton = append(rowsButton, inlineKeyboardButton10)

			inlineKeyboardButton11 = append(inlineKeyboardButton11, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_BirdsFeeding.String(), string(cons.CONTEST_BirdsFeeding)))
			rowsButton = append(rowsButton, inlineKeyboardButton11)

			inlineKeyboardButton12 = append(inlineKeyboardButton12, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Shrovetide.String(), string(cons.CONTEST_Shrovetide)))
			rowsButton = append(rowsButton, inlineKeyboardButton12)

			inlineKeyboardButton13 = append(inlineKeyboardButton13, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_Fable.String(), string(cons.CONTEST_Fable)))
			rowsButton = append(rowsButton, inlineKeyboardButton13)

			inlineKeyboardButton14 = append(inlineKeyboardButton14, tgbotapi.NewInlineKeyboardButtonData(cons.CONTEST_DefendersFatherland.String(), string(cons.CONTEST_DefendersFatherland)))
			rowsButton = append(rowsButton, inlineKeyboardButton14)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.COMPLETE_APPLICATION: %w", err)
			}
		}

	case botcommand.SELECT_PROJECT:

		if !thisIsAdmin(id) {

			msg := tgbotapi.NewMessage(id, message, styleText)

			msg.ReplyMarkup = keyboardApplicationStart

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_PROJECT: %w", err)
			}

			body := make([]string, 3)
			body = append(body, "В любой момент вы можете отменить заявку, нажав \"Отмена\"")
			body = append(body, "")
			body = append(body, fmt.Sprintf("Для продолжения заполнения заявки, необходимо дать согласие на обработку персональных данных и нажать \"Продолжить\".\n Ознакомиться с пользователським соглашением и политикой конфидециальности\n можно по ссылке %s", os.Getenv("PrivacyPolicy_Terms_Conditions")))
			text := strings.Join(body, "\n")

			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.AGREE), cons.AGREE.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg = tgbotapi.NewMessage(id, text, cons.StyleTextCommon)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_PROJECT: %w", err)
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
			return fmt.Errorf("sentToTelegram(), botcommand.SELECT_DEGREE: %w", err)
		}

	case botcommand.GET_DIPLOMA:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GET_DIPLOMA: %w", err)
		}

	case botcommand.GET_PUBLICATION_DATE:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GET_PUBLICATION_DATE: %w", err)
		}

	case botcommand.GET_PUBLICATION_LINK:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GET_PUBLICATION_LINK: %w", err)
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
				return fmt.Errorf("sentToTelegram(), botcommand.WAITING_FOR_ACCEPTANCE: %w", err)
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
			return fmt.Errorf("sentToTelegram(), botcommand.CONTINUE_DATA_POLLING: %w", err)
		}

	case botcommand.RECORD_TO_DB:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.RECORD_TO_DB: %w", err)
		}

	case botcommand.SELECT_FNP_LEADER:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling2
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SELECT_FNP_LEADER: %w", err)
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
			return fmt.Errorf("sentToTelegram(), botcommand.SELECT_DOCUMENT_TYPE: %w", err)
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
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_PLACE_DELIVERY_OF_DOCUMENTS: %w", err)
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
			return fmt.Errorf("sentToTelegram(), botcommand.CHECK_DATA: %w", err)
		}

		message = UserDataToStringForTelegramm(id)

		msg = tgbotapi.NewMessage(id, message, cons.StyleTextHTML)

		if !thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirm
		} else {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CHECK_DATA: %w", err)
		}

	case botcommand.CHECK_DATA_PAUSE:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CHECK_DATA_PAUSE: %w", err)
		}

	case botcommand.CHECK_PDF_FILES:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardConfirmAndSendForAdmin

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CHECK_PDF_FILES: %w", err)
		}

	case botcommand.UNDEFINED:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.UNDEFINED: %w", err)
		}

	case botcommand.SETTINGS:

	}

	return nil

}

func sentToTelegramPDF(bot *tgbotapi.BotAPI, id int64, pdf_path string, file_id string, command botcommand.BotCommand) error {

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
			return fmt.Errorf("sentToTelegramPDF(), botcommand.SELECT_PROJECT: %w", err)
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
			return fmt.Errorf("sentToTelegramPDF(), botcommand.SELECT_PROJECT: %w", err)
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

	userData := userPolling.Get(userID)

	row, err := dbpool.Query(ctx, fmt.Sprintf("insert into %s (user_id, contest, user_fnp, user_age, group_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, document_type, place_delivery_of_document, diploma, start_date, expiration, close_date) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) returning requisition_number", userData.TableDB), userID, userData.Contest, userData.FNP, userData.Age, userData.GroupAge, userData.NameInstitution, userData.Locality, userData.NamingUnit, userData.PublicationTitle, userData.LeaderFNP, userData.Email, userData.DocumentType, userData.PlaceDeliveryDocuments, userData.Diploma, time.Now().UnixNano(), int64(time.Now().Add(172800*time.Second).UnixNano()), 0)

	if err != nil {
		return fmt.Errorf("func AddRequisition(), query to db is failed: %w", err)
	}

	if row.Next() {

		var requisition_number int

		err := row.Scan(&requisition_number)

		if err != nil {
			return fmt.Errorf("func AddRequisition(), scan datas of row is failed %w", err)
		}

		userPolling.Set(userID, enumapplic.REQUISITION_NUMBER, fmt.Sprintf("%v", requisition_number))

		if userData.Diploma {

			var diploma_number int

			row, err := dbpool.Query(ctx, "insert into diplomas (requisition_number) values ($1) returning diploma_number", requisition_number)

			if err != nil {
				return fmt.Errorf("func AddRequisition(), query to db is failed: %w", err)
			}

			if row.Next() {

				err := row.Scan(&diploma_number)

				if err != nil {
					return fmt.Errorf("func AddRequisition(), scan datas of row is failed %w", err)
				}

				userPolling.Set(userID, enumapplic.DIPLOMA_NUMBER, fmt.Sprintf("%v", diploma_number))
			}

		}
	}

	return row.Err()
}

func GetRequisitionForAdmin(userPolling cache.CacheDataPolling, requisition_number int64, tableDB string, degree string, publicationDate, publicationLink string, dbpool *pgxpool.Pool, ctx context.Context) (err error, userID int64) {

	var fnp string
	var age int
	var group_age string
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
	var place_delivery_of_document string

	row, err := dbpool.Query(ctx, fmt.Sprintf("SELECT user_id, user_fnp, user_age, COALESCE(group_age, ''), name_institution, locality, naming_unit, publication_title, leader_fnp, email, contest, document_type, place_delivery_of_document, diploma, COALESCE(diploma_number, 0) FROM %s LEFT JOIN diplomas ON %s.requisition_number=diplomas.requisition_number WHERE %s.requisition_number = $1", tableDB, tableDB, tableDB), requisition_number)

	if err != nil {
		return fmt.Errorf("func GetRequisitionForAdmin(), query to db is failed: %w", err), 0
	}

	if row.Next() {

		err = row.Scan(&userID, &fnp, &age, &group_age, &name_institution, &locality, &naming_unit, &publication_title, &leader_fnp, &email, &contest, &document_type, &place_delivery_of_document, &diploma, &diploma_number)

		if err != nil {
			return fmt.Errorf("func GetRequisitionForAdmin(), scan datas of row is failed %w", err), 0
		}

		// When RequisitionNumber == 0 and Contest != ""  - most likely the user is working on a new requisition
		if userPolling.Get(userID).RequisitionNumber == 0 && userPolling.Get(userID).Contest != "" {
			tempUsersIDCache.Add(userID)
			err = &errs.ErrCacheUserPolling{UserID: userID, RequisitionNumber: requisition_number}
			return errs.ErrorHandler(err), userID
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
		userPolling.Set(userID, enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, place_delivery_of_document)
		userPolling.Set(userID, enumapplic.REQUISITION_NUMBER, fmt.Sprintf("%v", requisition_number))
		userPolling.Set(userID, enumapplic.PUBLICATION_LINK, publicationLink)
		userPolling.Set(userID, enumapplic.PUBLICATION_DATE, publicationDate)
		userPolling.Set(userID, enumapplic.DEGREE, degree)
		userPolling.Set(userID, enumapplic.TableDB, cons.CERTIFICATE.String())
		userPolling.Set(userID, enumapplic.DIPLOMA, strconv.FormatBool(diploma))
		if group_age != "" {
			userPolling.Set(userID, enumapplic.GROUP, "")
			userPolling.Set(userID, enumapplic.GROUP_AGE, group_age)
		}

		if diploma {
			userPolling.Set(userID, enumapplic.DIPLOMA_NUMBER, strconv.Itoa(int(diploma_number)))
		}

	}

	return row.Err(), userID
}

func GetRequisitionForUser(user_id int64, requisition_number int64, dbpool *pgxpool.Pool, ctx context.Context) (err error, userID int64, sent bool) {

	var fnp string
	var age int
	var group_age string
	var name_institution string
	var locality string
	var naming_unit string
	var publication_title string
	var publication_link string
	var publication_date int64
	var degree int
	var leader_fnp string
	var email string
	var contest string
	var document_type string
	var diploma bool
	var diploma_number int64

	row, err := dbpool.Query(ctx, fmt.Sprintf("SELECT user_id, user_fnp, user_age, COALESCE(group_age, ''), name_institution, locality, naming_unit, publication_title, COALESCE(reference, ''), publication_date, degree, leader_fnp, email, contest, document_type, diploma, COALESCE(diploma_number, 0) FROM %s LEFT JOIN diplomas ON %s.requisition_number=diplomas.requisition_number WHERE %s.requisition_number = $1", cons.CERTIFICATE.String(), cons.CERTIFICATE.String(), cons.CERTIFICATE.String()), requisition_number)

	if err != nil {
		return fmt.Errorf("func GetRequisitionForUser(), query to db is failed: %W", err), 0, sent
	}

	if row.Next() {

		err = row.Scan(&userID, &fnp, &age, &group_age, &name_institution, &locality, &naming_unit, &publication_title, &publication_link, &publication_date, &degree, &leader_fnp, &email, &contest, &document_type, &diploma, &diploma_number)

		if err != nil {
			return fmt.Errorf("func GetRequisitionForUser(), scan datas of row is failed %w", err), 0, sent
		}

		if userID == 0 && publication_date != 0 {
			sent = true
		}

		if userID == 0 || userID != user_id {
			return row.Err(), userID, sent
		}

		dateString := unixNanoToDateString(publication_date)

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
		userPolling.Set(userID, enumapplic.PUBLICATION_LINK, publication_link)
		userPolling.Set(userID, enumapplic.PUBLICATION_DATE, dateString)
		userPolling.Set(userID, enumapplic.DEGREE, strconv.Itoa(degree))
		userPolling.Set(userID, enumapplic.TableDB, cons.CERTIFICATE.String())
		userPolling.Set(userID, enumapplic.DIPLOMA, strconv.FormatBool(diploma))
		if group_age != "" {
			userPolling.Set(userID, enumapplic.GROUP, "")
			userPolling.Set(userID, enumapplic.GROUP_AGE, group_age)
		}
		if diploma {
			userPolling.Set(userID, enumapplic.DIPLOMA_NUMBER, strconv.Itoa(int(diploma_number)))
		}

	}

	return row.Err(), userID, sent
}

func UpdateRequisition(admin bool, cleanOut bool, requisition_number int64, tableDB string, degree int, publicationLink string, publication_date string, dbpool *pgxpool.Pool, ctx context.Context) (err error) {

	var query string

	publicationDate := dateStringToUnixNano(publication_date)

	switch admin {

	case true:

		if cleanOut {
			query = fmt.Sprintf("UPDATE %s SET reference='%v', publication_date='%v', close_date='%v', degree='%v', email='%v',user_fnp='%v',leader_fnp='%v',user_id='%v' WHERE requisition_number=$1 RETURNING user_id", tableDB, publicationLink, publicationDate, time.Now().UnixNano(), degree, "", "", "", 0)
		} else {
			query = fmt.Sprintf("UPDATE %s SET reference='%v', publication_date='%v', close_date='%v', degree='%v' WHERE requisition_number=$1 RETURNING user_id", tableDB, publicationLink, publicationDate, time.Now().UnixNano(), degree)
		}

	default:
		query = fmt.Sprintf("UPDATE %s SET email='%v',user_fnp='%v',leader_fnp='%v',user_id='%v' WHERE requisition_number=$1 RETURNING user_id", tableDB, "", "", "", 0)
	}

	row, err := dbpool.Query(ctx, query, requisition_number)

	if err != nil {
		return fmt.Errorf("query UPDATE to db is failed: %w", err)
	}

	row.Next()

	return row.Err()
}

func FillInCertificatesPDFForms(wg *sync.WaitGroup, userID int64, userPolling cache.CacheDataPolling) {

	defer wg.Done()

	var x float64
	var y float64 = 305
	var step float64 = 15
	var widthText float64
	var centerX float64 = 297.5
	var path string
	var degree string

	usersRequisition := userPolling.Get(userID)

	if usersRequisition.LeaderFNP != "" {

		path = "./external/imgs/%s_%s_curator.jpg"

	} else {

		path = "./external/imgs/%s_%s.jpg"

	}

	boilerplatePDFPath := fmt.Sprintf(path, contests[usersRequisition.Contest], cons.CERTIFICATE.String())

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), AddTTFFont: %v", err.Error()))
	}

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	err = pdf.Image(boilerplatePDFPath, 0, 0, rect)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Image(): %v", err))
	}

	//1. Degree

	pdf.SetXY(235, 211)
	pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
	err = pdf.SetFont("TelegraphLine", "", 24)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), SetFont: %v", err.Error()))
	}

	switch usersRequisition.Degree {

	case 1:
		degree = "I"
	case 2:
		degree = "II"
	case 3:
		degree = "III"
	}

	err = pdf.Text(fmt.Sprintf("%s", degree))

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(), degree: %v", err.Error()))
	}

	//2. Requisition number

	switch {
	case usersRequisition.RequisitionNumber > 10000:
		x = 81
	case usersRequisition.RequisitionNumber > 1000:
		x = 91
	case usersRequisition.RequisitionNumber > 100:
		x = 101
	default:
		x = 111
	}

	pdf.SetXY(x, 251)
	pdf.SetTextColorCMYK(58, 46, 41, 94) //black
	err = pdf.SetFont("TelegraphLine", "", 14)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont(): %v", err.Error()))
	}

	err = pdf.Text(fmt.Sprintf("%v", usersRequisition.RequisitionNumber))

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(): %v", err.Error()))
	}

	//3. Name

	pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
	err = pdf.SetFont("TelegraphLine", "", 22)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont(): %v", err.Error()))
	}

	widthText, err = pdf.MeasureTextWidth(usersRequisition.FNP)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
	}

	x = centerX - widthText/2

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(usersRequisition.FNP, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(): %v", err.Error()))
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
		pdf.SetXY(x, 275)
		err = pdf.Text(usersRequisition.FNP)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.FNP): %v", err.Error()))
		}
	}

	//4. Age
	var age_string string
	var ending string

	switch usersRequisition.Group {

	case true:
		groupAge := strings.TrimSpace(usersRequisition.GroupAge)
		age_string = groupAge

		if groupAge != "0" {
			var symbol string

			contain1 := strings.Contains(groupAge, "лет")
			contain2 := strings.Contains(groupAge, "года")
			contain3 := strings.Contains(groupAge, "год")
			contain4 := strings.Contains(groupAge, "годов")
			contain5 := strings.Contains(groupAge, "годиков")
			contain6 := strings.Contains(groupAge, "годков")

			if !contain1 && !contain2 && !contain3 && !contain4 && !contain5 && !contain6 {
				var age int
				for i := 1; i < len(groupAge)-1; i++ {
					symbol = string(groupAge[len(groupAge)-i:])
					age, err = strconv.Atoi(symbol)

					if err != nil {
						symbol = string(groupAge[len(groupAge)-i+1:])
						age, _ = strconv.Atoi(symbol)
						break
					}
				}
				ending = convertAgeToString(age)
				age_string = fmt.Sprintf("%v %v", age_string, ending)
			}
		}

	case false:
		ending = convertAgeToString(usersRequisition.Age)
		age_string = fmt.Sprintf("%v %v", usersRequisition.Age, ending)
	}

	widthText, err = pdf.MeasureTextWidth(age_string)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(age_string): %v", err))
	}

	x = centerX - widthText/2
	y = pdf.GetY() + 1.5*step

	pdf.SetXY(x, y)
	err = pdf.Text(age_string)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(): %v", err))
	}

	//5. Name institution

	y = pdf.GetY() + 2*step

	pdf.SetTextColorCMYK(58, 46, 41, 94) //black
	err = pdf.SetFont("TelegraphLine", "", 18)

	widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err))
	}

	if widthText > maxWidthPDF-80 {

		arrayText := strings.Split(usersRequisition.NameInstitution, " ")

		var t string

		for _, word := range arrayText {

			t = fmt.Sprintf("%s %s", t, word)

			widthText, err = pdf.MeasureTextWidth(t)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
			}

			if widthText > maxWidthPDF-80 {

				widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution[:len(t)-1])

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
				}

				x = centerX - widthText/2

				textPart1 := usersRequisition.NameInstitution[:len(t)-1]

				pdf.SetXY(x, y)
				pdf.Text(textPart1)
				y = y + step

				textPart2 := usersRequisition.NameInstitution[len(t)-1:]

				widthText, err = pdf.MeasureTextWidth(textPart2)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				pdf.Text(textPart2)
				y = y + step

				zrlog.Info().Msg("Split long name institution")

				break

			}
		}

	} else {

		y = pdf.GetY() + 2*step
		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(usersRequisition.NameInstitution)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.NameInstitution): %v", err.Error()))
		}
	}

	//6. Locality

	y = pdf.GetY() + 1.5*step

	pdf.SetTextColorCMYK(58, 46, 41, 94) //black

	widthText, err = pdf.MeasureTextWidth(usersRequisition.Locality)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality): %v", err.Error()))
	}

	if widthText > maxWidthPDF-80 {

		arrayText := strings.Split(usersRequisition.Locality, " ")

		var t string

		for _, word := range arrayText {

			t = fmt.Sprintf("%s %s", t, word)

			widthText, err = pdf.MeasureTextWidth(t)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
			}

			if widthText > maxWidthPDF-80 {

				textPart1 := usersRequisition.Locality[:len(t)-1]

				widthText, err = pdf.MeasureTextWidth(textPart1)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality[:len(t)]): %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				pdf.Text(textPart1)
				y = y + step

				textPart2 := usersRequisition.Locality[len(t)-1:]

				widthText, err = pdf.MeasureTextWidth(textPart2)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality[:len(t)]): %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				pdf.Text(textPart2)
				y = y + step
				break
			}
		}

	} else {

		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(usersRequisition.Locality)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.Locality): %v", err.Error()))
		}
	}

	//7. Naming unit

	pdf.SetXY(152, 622)
	pdf.SetTextColorCMYK(58, 46, 41, 94) //black

	err = pdf.Text(usersRequisition.NamingUnit)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.NamingUnit): %v", err.Error()))
	}

	//8. Publication title

	pdf.SetXY(194, 646)
	pdf.SetTextColorCMYK(58, 46, 41, 94) //black

	err = pdf.Text(usersRequisition.PublicationTitle)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.PublicationTitle): %v", err.Error()))
	}

	//9. Leader's FNP

	if usersRequisition.LeaderFNP != "" {

		y = 668
		x = 194

		pdf.SetXY(x, y)

		var arrayLeaders []string
		var maxWidth float64

		contain := strings.Contains(usersRequisition.LeaderFNP, cons.Comma)

		switch contain {

		case true:

			arrayLeaders = strings.Split(usersRequisition.LeaderFNP, cons.Comma)

			for i, leader := range arrayLeaders {

				if i == 0 {
					leader = fmt.Sprintf("%s,", strings.TrimSpace(leader))
					maxWidth = (maxWidthPDF + 225) / 2
				} else {
					leader = strings.TrimSpace(leader)
					y = pdf.GetY() + 1.2*step
					maxWidth = maxWidthPDF
				}

				widthText, err = pdf.MeasureTextWidth(leader)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(leader): %v", err.Error()))
				}

				if widthText > maxWidth {

					var arrayText []string

					arrayText, err = pdf.SplitText(leader, maxWidth)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(leader, maxWidth): %v", err.Error()))
					}

					for k, t := range arrayText {

						widthText, err = pdf.MeasureTextWidth(t)

						if i > 0 || k > 0 { //Second leader or second part part first leader
							x = 55
						}

						pdf.SetXY(x, y)
						pdf.Text(t)
						y = y + 1.2*step
					}

				} else {

					if i > 0 { //Second leader
						x = 55
					}

					pdf.SetXY(x, y)

					err = pdf.Text(leader)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(leader): %v", err.Error()))
					}

					y = y + 1.2*step
				}

			}

		case false:

			maxWidth = (maxWidthPDF + 225) / 2
			widthText, err = pdf.MeasureTextWidth(usersRequisition.LeaderFNP)

			if widthText > maxWidth {

				var arrayText []string

				arrayText, err = pdf.SplitText(usersRequisition.LeaderFNP, maxWidth)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(usersRequisition.LeaderFNP, maxWidth): %v", err.Error()))
				}

				for k, t := range arrayText {

					if k > 0 {
						x = 55
					}

					pdf.SetXY(x, y)
					pdf.Text(t)
					y = y + 1.2*step
				}

			} else {

				pdf.SetXY(x, y)
				err = pdf.Text(usersRequisition.LeaderFNP)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.LeaderFNP): %v", err.Error()))
				}
			}
		}
	}

	//10. Publication date

	pdf.SetXY(426, 718)
	pdf.SetTextColorCMYK(58, 46, 41, 94) //black

	err = pdf.Text(usersRequisition.PublicationDate)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.PublicationDate): %v", err.Error()))
	}

	//11. Publication link

	pdf.SetXY(50, 740)
	pdf.SetTextColorCMYK(58, 46, 41, 94) //black

	err = pdf.Text(usersRequisition.PublicationLink)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(usersRequisition.PublicationLink): %v", err.Error()))
	}

	path = fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.CERTIFICATE), usersRequisition.RequisitionNumber)

	err = pdf.WritePdf(path)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.WritePdf(path): %v", err.Error()))
	}

	userPolling.Set(userID, enumapplic.FILE, path)

	err = pdf.Close()

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(),  pdf.Close(): %v", err.Error()))
	}

}

func FillInDiplomasPDFForms(wg *sync.WaitGroup, userID int64, userPolling cache.CacheDataPolling) {

	defer wg.Done()

	var x float64
	var y float64 = 305
	var step float64 = 15
	var widthText float64
	var centerX float64 = 297.5
	var degree string

	usersRequisition := userPolling.Get(userID)

	if usersRequisition.Diploma {

		boilerplatePDFPath := fmt.Sprintf("./external/imgs/%s_%s.jpg", contests[usersRequisition.Contest], cons.DIPLOMA.String())

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

		err := pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.AddTTFFont(): %v", err.Error()))
		}

		pdf.AddPage()
		rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
		err = pdf.Image(boilerplatePDFPath, 0, 0, rect)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Image(boilerplatePDFPath, 0, 0, rect): %v", err.Error()))
		}

		//1. Diploma number

		pdf.SetTextColorCMYK(58, 46, 41, 94) //black
		err = pdf.SetFont("TelegraphLine", "", 14)

		x = 90

		pdf.SetXY(x, 242)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		err = pdf.Text(fmt.Sprintf("%v", usersRequisition.DiplomaNumber))

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.DiplomaNumber): %v", err.Error()))
		}

		//2. Leader's FNP
		pdf.SetTextColorCMYK(0, 100, 100, 0) //Red
		err = pdf.SetFont("TelegraphLine", "", 18)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		var arrayLeaders []string

		y = 262

		contain := strings.Contains(usersRequisition.LeaderFNP, cons.Comma)

		switch contain {

		case true:

			arrayLeaders = strings.Split(usersRequisition.LeaderFNP, cons.Comma)

			for i, leader := range arrayLeaders {

				if i == 0 {
					leader = fmt.Sprintf("%s,", strings.TrimSpace(leader))
				} else {
					leader = strings.TrimSpace(leader)
					y = pdf.GetY() + 1.3*step
				}

				widthText, err = pdf.MeasureTextWidth(leader)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(leader): %v", err.Error()))
				}

				x = centerX - widthText/2

				if widthText > maxWidthPDF {

					var arrayText []string

					arrayText, err = pdf.SplitText(leader, maxWidthPDF)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SplitText(leader, maxWidthPDF): %v", err.Error()))
					}

					for _, t := range arrayText {

						widthText, err = pdf.MeasureTextWidth(t)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
						}

						x = centerX - widthText/2

						pdf.SetXY(x, y)

						pdf.Text(t)
						y = y + 1.2*step
					}

				} else {

					pdf.SetXY(x, y)
					err = pdf.Text(leader)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(leader): %v", err.Error()))
					}

				}

			}

		case false:

			if widthText > maxWidthPDF {

				var arrayText []string

				arrayText, err = pdf.SplitText(usersRequisition.LeaderFNP, maxWidthPDF)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SplitText(usersRequisition.LeaderFNP, maxWidthPDF): %v", err.Error()))
				}

				y = pdf.GetY() + 2*step

				for _, t := range arrayText {

					widthText, err = pdf.MeasureTextWidth(t)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					pdf.Text(t)
					y = y + 1.2*step
				}

			} else {

				widthText, err = pdf.MeasureTextWidth(usersRequisition.LeaderFNP)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.LeaderFNP): %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				err = pdf.Text(usersRequisition.LeaderFNP)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.LeaderFNP): %v", err.Error()))
				}
			}
		}

		//3. Name institution

		pdf.SetTextColorCMYK(58, 46, 41, 94) //black
		err = pdf.SetFont("TelegraphLine", "", 16)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), Name institution, pdf.SetFont(): %v", err.Error()))
		}

		y = pdf.GetY() + 1.5*step

		widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.NameInstitution): %v", err.Error()))
		}

		if widthText > maxWidthPDF-20 {

			arrayText := strings.Split(usersRequisition.NameInstitution, " ")

			var t string

			for _, word := range arrayText {

				t = fmt.Sprintf("%s %s", t, word)

				widthText, err = pdf.MeasureTextWidth(t)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
				}

				if widthText > maxWidthPDF-20 {

					widthText, err = pdf.MeasureTextWidth(usersRequisition.NameInstitution[:len(t)])

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.NameInstitution[:len(t)]): %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					pdf.Text(usersRequisition.NameInstitution[:len(t)])
					y = y + step

					pdf.SetXY(x, y)
					pdf.Text(usersRequisition.NameInstitution[len(t):])
					y = y + step
					break

				}
			}

		} else {

			y = pdf.GetY() + 2*step
			x = centerX - widthText/2

			pdf.SetXY(x, y)
			err = pdf.Text(usersRequisition.NameInstitution)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.NameInstitution): %v", err.Error()))
			}
		}

		//4. Locality
		pdf.SetTextColorCMYK(58, 46, 41, 94) //black
		err = pdf.SetFont("TelegraphLine", "", 16)

		y = pdf.GetY() + 1.5*step

		widthText, err = pdf.MeasureTextWidth(usersRequisition.Locality)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality): %v", err.Error()))
		}

		if widthText > maxWidthPDF-20 {

			arrayText := strings.Split(usersRequisition.Locality, " ")

			var t string

			for _, word := range arrayText {

				t = fmt.Sprintf("%s %s", t, word)

				widthText, err = pdf.MeasureTextWidth(t)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
				}

				if widthText > maxWidthPDF-20 {

					widthText, err = pdf.MeasureTextWidth(usersRequisition.Locality[:len(t)])

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality[:len(t)]): %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					pdf.Text(usersRequisition.Locality[:len(t)])
					y = y + step

					pdf.SetXY(x, y)
					pdf.Text(usersRequisition.Locality[len(t):])
					y = y + step
					break
				}
			}

		} else {

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			err = pdf.Text(usersRequisition.Locality)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.Locality): %v", err.Error()))
			}
		}

		//5. FNP
		pdf.SetXY(142, 627)
		err = pdf.SetFont("TelegraphLine", "", 18)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		err = pdf.Text(usersRequisition.FNP)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.FNP): %v", err.Error()))
		}

		//6. Naming unit

		pdf.SetXY(153, 653)

		err = pdf.Text(usersRequisition.NamingUnit)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.NamingUnit): %v", err.Error()))
		}

		//7. Publication title

		pdf.SetXY(195, 674)

		err = pdf.Text(usersRequisition.PublicationTitle)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationTitle): %v", err.Error()))
		}

		//8. Requisition number

		pdf.SetXY(139, 697)

		err = pdf.Text(fmt.Sprintf("%v", usersRequisition.RequisitionNumber))

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.RequisitionNumber): %v", err.Error()))
		}

		//9. Degree

		var textDegree string

		switch usersRequisition.Degree {

		case 1:
			degree = "I"
			textDegree = fmt.Sprintf(",   %s", degree)
		case 2:
			degree = "II"
			x = 230
			textDegree = fmt.Sprintf(",  %s", degree)
		case 3:
			degree = "III"
			textDegree = fmt.Sprintf(", %s", degree)
		}

		pdf.SetXY(228, 717)
		err = pdf.Text(textDegree)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(textDegree): %v", err.Error()))
		}

		//10. Publication date

		pdf.SetXY(447, 736)

		err = pdf.Text(usersRequisition.PublicationDate)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationDate): %v", err.Error()))
		}

		//11. Publication link

		pdf.SetXY(56, 757)

		err = pdf.Text(usersRequisition.PublicationLink)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationLink): %v", err.Error()))
		}

		path := fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.DIPLOMA), usersRequisition.RequisitionNumber)

		err = pdf.WritePdf(path)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.WritePdf(path): %v", err.Error()))
		}

		userPolling.Set(userID, enumapplic.FILE, path)

		err = pdf.Close()

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Close(): %v", err.Error()))
		}
	}
}

func ConvertRequisitionToPDF(userID int64) (bool, error) {

	usersRequisition := userPolling.Get(userID)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := pdf.AddTTFFont("Merriweather-Bold", "./external/fonts/ttf/Merriweather-Bold.ttf")

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.AddTTFFont(): %v", err.Error()))
	}

	pdf.SetTextColorCMYK(100, 100, 100, 100)

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} //Page size A4 format
	err = pdf.Image("./external/imgs/RequisitionsBoilerplate.jpg", 0, 0, rect)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Image(): %v", err.Error()))
	}

	pdf.SetXY(200, 220)
	pdf.SetTextColorCMYK(100, 70, 0, 67)
	err = pdf.SetFont("Merriweather-Bold", "", 14)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.SetFont(): %v", err.Error()))
	}

	t := time.Now()
	formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

	err = pdf.CellWithOption(nil, fmt.Sprintf("Заявка №%v от %v ", usersRequisition.RequisitionNumber, formattedTime), cellOption_Caption)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.CellWithOption(): %v", err.Error()))
	}

	y := 270.0
	step := 30.0

	pdf.SetXY(25, y)
	err = pdf.Text(fmt.Sprintf("Участник: %s", usersRequisition.FNP))
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(usersRequisition.FNP): %v", err.Error()))
	}

	y = y + step
	pdf.SetXY(25, y)
	if usersRequisition.LeaderFNP != "" {
		err = pdf.Text(fmt.Sprintf("Руководитель: %s", usersRequisition.LeaderFNP))
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(usersRequisition.LeaderFNP): %v", err.Error()))
		}
		y = y + step
	}

	pdf.SetXY(25, y)

	err = pdf.Text(fmt.Sprintf("Конкурс: \"%s\"", usersRequisition.Contest))
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(usersRequisition.Contest): %v", err.Error()))
	}
	y = y + step

	pdf.SetXY(25, y)
	text := fmt.Sprintf("%s: \"%s\"", enumapplic.NAMING_UNIT, usersRequisition.NamingUnit)
	widthText, err := pdf.MeasureTextWidth(text)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.MeasureTextWidth(text): %v", err.Error()))
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.SplitText(text, maxWidthPDF): %v", err.Error()))
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(text): %v", err.Error()))
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: \"%s\"", enumapplic.PUBLICATION_TITLE, usersRequisition.PublicationTitle)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.MeasureTextWidth(text): %v", err.Error()))
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.SplitText(text, maxWidthPDF): %v", err.Error()))
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(text): %v", err.Error()))
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: %s", enumapplic.DOCUMENT_TYPE, usersRequisition.DocumentType)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.MeasureTextWidth(text): %v", err.Error()))
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.SplitText(text, maxWidthPDF): %v", err.Error()))
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(text): %v", err.Error()))
		}
		y = y + step
	}

	pdf.SetXY(25, y)
	text = fmt.Sprintf("%s: %s", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS, usersRequisition.PlaceDeliveryDocuments)
	widthText, err = pdf.MeasureTextWidth(text)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.MeasureTextWidth(text): %v", err.Error()))
	}

	if widthText > maxWidthPDF {

		var arrayText []string

		arrayText, err = pdf.SplitText(text, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.SplitText(text, maxWidthPDF): %v", err.Error()))
		}

		for _, t := range arrayText {
			pdf.SetXY(25, y)
			pdf.Text(t)
			y = y + step
		}

	} else {
		err = pdf.Text(text)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(),  pdf.Text(text): %v", err.Error()))
		}
		y = y + step
	}

	err = pdf.WritePdf(fmt.Sprintf("./external/files/usersfiles/Заявка_№%v.pdf", usersRequisition.RequisitionNumber))

	if err != nil {
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

		temp := ""
		for _, path := range files {

			//When some files are the same
			if temp != "" && temp == path {
				continue
			}

			m.Attach(path)

			temp = path
		}
	}

	// Set the email body. You can set plain text or html with text/html
	m.SetBody("text/html", message)

	// Settings for SMTP server
	d := gomail.NewDialer(os.Getenv("SMTP_SERVER"), 465, os.Getenv("BOT_LOGIN_EMAIL"), os.Getenv("BOT_PASSWORD_EMAIL"))

	if err := d.DialAndSend(m); err != nil {
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

	var age_string string
	if (!usdata.Group && usdata.Age != 0) || (usdata.Group && strings.TrimSpace(usdata.GroupAge) != "0") {
		var ending string
		switch usdata.Group {
		case true:
			groupAge := strings.TrimSpace(usdata.GroupAge)
			age_string = groupAge

			if groupAge != "0" {
				var symbol string

				contain1 := strings.Contains(groupAge, "лет")
				contain2 := strings.Contains(groupAge, "года")
				contain3 := strings.Contains(groupAge, "год")

				if !contain1 && !contain2 && !contain3 {
					var age int
					var err error

					for i := 1; i < len(groupAge)-1; i++ {
						symbol = string(groupAge[len(groupAge)-i:])
						age, err = strconv.Atoi(symbol)

						if err != nil {
							symbol = string(groupAge[len(groupAge)-i+1:])
							age, err = strconv.Atoi(symbol)
							break
						}
					}
					ending = convertAgeToString(age)
					age_string = fmt.Sprintf("%v %v", age_string, ending)
				}
			}

		case false:
			ending = convertAgeToString(usdata.Age)
			age_string = fmt.Sprintf("%v %v", usdata.Age, ending)
		}

	} else {
		age_string = "возраст не будет указан в грамоте/дипломе"
	}
	body = append(body, fmt.Sprintf("<dd><p>      %v</p></dd>", age_string))
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

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.CONTEST.EnumIndex(), enumapplic.CONTEST.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Contest))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.FNP))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.AGE.EnumIndex(), enumapplic.AGE.String()))

		var age_string string

		if (!usdata.Group && usdata.Age != 0) || (usdata.Group && strings.TrimSpace(usdata.GroupAge) != "0") {

			var ending string
			switch usdata.Group {
			case true:
				groupAge := strings.TrimSpace(usdata.GroupAge)
				age_string = groupAge

				if groupAge != "0" {
					var symbol string

					contain1 := strings.Contains(groupAge, "лет")
					contain2 := strings.Contains(groupAge, "года")
					contain3 := strings.Contains(groupAge, "год")
					contain4 := strings.Contains(groupAge, "годов")
					contain5 := strings.Contains(groupAge, "годиков")
					contain6 := strings.Contains(groupAge, "годков")

					if !contain1 && !contain2 && !contain3 && !contain4 && !contain5 && !contain6 {
						var age int
						var err error

						for i := 1; i < len(groupAge)-1; i++ {
							symbol = string(groupAge[len(groupAge)-i:])
							age, err = strconv.Atoi(symbol)

							if err != nil {
								symbol = string(groupAge[len(groupAge)-i+1:])
								age, _ = strconv.Atoi(symbol)
								break
							}
						}
						ending = convertAgeToString(age)
						age_string = fmt.Sprintf("%v %v", age_string, ending)
					}
				}

			case false:
				ending = convertAgeToString(usdata.Age)
				age_string = fmt.Sprintf("%v %v", usdata.Age, ending)
			}

		} else {
			age_string = "возраст не будет указан в грамоте/дипломе"
		}

		body = append(body, fmt.Sprintf("      %v", age_string))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NAME_INSTITUTION.EnumIndex(), enumapplic.NAME_INSTITUTION.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.NameInstitution))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.LOCALITY.EnumIndex(), enumapplic.LOCALITY.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Locality))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NAMING_UNIT.EnumIndex(), enumapplic.NAMING_UNIT.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.NamingUnit))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PUBLICATION_TITLE.EnumIndex(), enumapplic.PUBLICATION_TITLE.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.PublicationTitle))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		if usdata.LeaderFNP == "" {
			body = append(body, fmt.Sprintf("(%v). <s><i><b>%s:</b></i></s>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
			body = append(body, fmt.Sprintf("      %s", "-"))
		} else {
			body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNP_LEADER.EnumIndex(), enumapplic.FNP_LEADER.String()))
			body = append(body, fmt.Sprintf("      %s", usdata.LeaderFNP))
		}
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.EMAIL.EnumIndex(), enumapplic.EMAIL.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Email))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.DOCUMENT_TYPE.EnumIndex(), enumapplic.DOCUMENT_TYPE.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.DocumentType))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.EnumIndex(), enumapplic.PLACE_DELIVERY_OF_DOCUMENTS.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.PlaceDeliveryDocuments))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PHOTO.EnumIndex(), enumapplic.PHOTO.String()))
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FILE.EnumIndex(), enumapplic.FILE.String()))
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")

	} else {

		data := closingRequisition.Get(userID)
		body := make([]string, 12)

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.REQUISITION_NUMBER.String()))
		body = append(body, fmt.Sprintf("   %v", data.RequisitionNumber))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.DEGREE.String()))
		body = append(body, fmt.Sprintf("   %s", data.Degree))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PUBLICATION_DATE.String()))
		body = append(body, fmt.Sprintf("   %s", data.PublicationDate))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PUBLICATION_LINK.String()))
		body = append(body, fmt.Sprintf("   %s", data.PublicationLink))
		text = strings.Join(body, "\n")
	}

	return text
}

func GetConciseDescription(contest string) string {

	var text string

	body := make([]string, 14)

	if contest == string(cons.CONTEST_Titmouse) || contest == string(cons.CONTEST_Mather) || contest == string(cons.CONTEST_Father) || contest == string(
		cons.CONTEST_Autumn) || contest == string(cons.CONTEST_Winter) || contest == string(cons.CONTEST_Snowflakes) || contest == string(cons.CONTEST_Snowman) || contest == string(
		cons.CONTEST_Symbol) || contest == string(cons.CONTEST_Heart) || contest == string(cons.CONTEST_Secrets) || contest == string(cons.CONTEST_BirdsFeeding) || contest == string(
		cons.CONTEST_Shrovetide) || contest == string(cons.CONTEST_Fable) || contest == string(cons.CONTEST_DefendersFatherland) {

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
			zrlog.Fatal().Msg(fmt.Sprintf("func getFile(), bot can't download this file: %+v\n", err.Error()))
		}

	}

}

func deleteUserPolling(userID int64, userData cache.CacheDataPolling) {

	userDP := userData.Get(userID)

	//delete user's files and datas in hashmap

	//removing file from the directory

	if userDP.RequisitionPDFpath != "" {
		e := os.Remove(userDP.RequisitionPDFpath)
		if e != nil {
			zrlog.Error().Msg(fmt.Sprintf("Error delete reqisition PDF file: %+v\n", e.Error()))
		}
	}

	if userDP.Photo != "" {
		e := os.Remove(userDP.Photo)
		if e != nil {
			zrlog.Error().Msg(fmt.Sprintf("func deleteUserPolling(), error delete file user's foto: %+v\n", e.Error()))
		}
	}

	temp := ""
	for _, path := range userDP.Files {

		//When some files are the same
		if temp != "" && temp == path {
			continue
		}

		e := os.Remove(path)
		if e != nil {
			zrlog.Error().Msg(fmt.Sprintf("func deleteUserPolling(), error delete user's files: %+v\n", e))
		}

		temp = path
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

	year, err := strconv.Atoi(y)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func dateStringToUnixNano(), year: %+v\n", err.Error()))
	}

	month, err := strconv.Atoi(m)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func dateStringToUnixNano(), month: %+v\n", err.Error()))
	}

	day, err := strconv.Atoi(d)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func dateStringToUnixNano(), day: %+v\n", err.Error()))
	}

	unixTime := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return unixTime.UnixNano()
}

func unixNanoToDateString(publication_date int64) string {

	t := time.Unix(0, publication_date)

	dateString := t.Format(cons.TimeshortForm)

	var d string
	var m string
	var y string

	sliceDate := strings.Split(dateString, "-")

	for k, v := range sliceDate {

		switch k {
		case 0:
			y = v
		case 1:
			d = v
		case 2:
			m = v
		}
	}

	return fmt.Sprintf("%s.%s.%s", d, m, y)

}

func checkUsersIDCache(userID int64, bot *tgbotapi.BotAPI) {

	if tempUsersIDCache.Check(userID) {

		tempUsersIDCache.Delete(userID)

		adminID, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)

		err = sentToTelegram(bot, adminID, fmt.Sprintf("Можно закрывать заявки для пользователя %v!", userID), nil, cons.StyleTextCommon, botcommand.UNDEFINED, "", "", false)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func checkUsersIDCache(), sentToTelegramm() for admin: %v\n", err))
		}

	}
}

func convertAgeToString(age int) string {

	var ending string

	age_string := strconv.Itoa(age)

	var numPrev string
	var numLast string
	var numbers [2]int

	if age >= 10 {
		numPrev = age_string[len(age_string)-2 : len(age_string)-1]
	} else {
		numPrev = "0"
	}

	numLast = age_string[len(age_string)-1:]

	numbers[0], _ = strconv.Atoi(numPrev)
	numbers[1], _ = strconv.Atoi(numLast)

	if age >= 10 {

		switch numbers[0] {
		case 1:
			ending = "лет"
		default:

			switch numbers[1] {
			case 0:
				if numbers[0] != 0 {
					ending = "лет"
				} else {
					ending = ""
				}
			case 1:
				ending = "год"
			case 2:
				ending = "года"
			case 3:
				ending = "года"
			case 4:
				ending = "года"
			default:
				ending = "лет"
			}

		}

	} else {

		switch numbers[1] {
		case 0:
			ending = ""
		case 1:
			ending = "год"
		case 2:
			ending = "года"
		case 3:
			ending = "года"
		case 4:
			ending = "года"
		default:
			ending = "лет"
		}
	}

	return ending
}
