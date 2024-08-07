package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"net/mail"
	"os"
	"path"
	"runtime/pprof"
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
	pgx "github.com/jackc/pgx/v5"
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
			tgbotapi.NewKeyboardButton(botcommand.CompleteApplication.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.GetDiploma.String()),
		),
	)

	keyboardApplicationStart = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Continue.String()),
			tgbotapi.NewKeyboardButton(botcommand.Cancel.String()),
		),
	)

	keyboardContinueClosingApplication = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Cancel.String()),
		),
	)

	keyboardContinueDataPolling1 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CancelApplication.String()),
		),
	)

	keyboardContinueDataPolling2 = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Down.String()),
			tgbotapi.NewKeyboardButton(botcommand.CancelApplication.String()),
		),
	)

	keyboardConfirm = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Confirm.String()),
			//tgbotapi.NewKeyboardButton(botcommand.SelectCorrection.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CancelApplication.String()),
		),
	)

	keyboardConfirmForAdmin = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Confirm.String()),
			tgbotapi.NewKeyboardButton(botcommand.CancelCloseRequisition.String()),
		),
	)

	keyboardConfirmAndSendForAdmin = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.SendPDFFiles.String()),
			tgbotapi.NewKeyboardButton(botcommand.CancelCloseRequisition.String()),
		),
	)

	keyboardAdminMainMenue = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.CloseRequisitionStart.String()),
		),

		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(botcommand.Settings.String()),
		),
	)

	contests = map[string]string{
		cons.ContestTitmouse.String():            "Titmouse",
		cons.ContestMather.String():              "Mather",
		cons.ContestFather.String():              "Father",
		cons.ContestAutumn.String():              "Autumn",
		cons.ContestWinter.String():              "Winter",
		cons.ContestSnowflakes.String():          "Snowflakes",
		cons.ContestSnowman.String():             "Snowman",
		cons.ContestSymbol.String():              "Symbol",
		cons.ContestHearts.String():              "Hearts",
		cons.ContestSecrets.String():             "Secrets",
		cons.ContestBirdsFeeding.String():        "BirdsFeeding",
		cons.ContestShrovetide.String():          "Shrovetide",
		cons.ContestFable.String():               "Fable",
		cons.ContestDefendersFatherland.String(): "DefendersFatherland",
		cons.ContestSpring.String():              "Spring",
		cons.ContestMarchEighth.String():         "MarchEighth",
		cons.ContestEarth.String():               "Earth",
		cons.ContestSpaceAdventures.String():     "SpaceAdventures",
		cons.ContestFeatheredFriends.String():    "FeatheredFriends",
		cons.ContestTheatricalBackstage.String(): "TheatricalBackstage",
		cons.ContestOurFriends.String():          "OurFriends",
		cons.ContestPrimroses.String():           "Primroses",
		cons.ContestVictoryDay.String():          "VictoryDay",
		cons.ContestMyFamily.String():            "MyFamily",
		cons.ContestMotherRussia.String():        "MotherRussia",
		cons.ContestChildProtectionDay.String():  "ChildProtectionDay",
		cons.ContestFire.String():                "Fire",
		cons.ContestTrafficLight.String():        "TrafficLight",
		cons.ContestSummerPalette.String():       "SummerPalette",
	}

	userPolling        = cache.NewCacheDataPolling()
	closingRequisition = cache.NewCacheDataClosingRequisition()
	cellOptionCaption  = gopdf.CellOption{Align: 16}

	wg sync.WaitGroup

	maxWidthPDF float64 = 507.0

	cacheBotSt cache.BotState
)

func main() {

	logFile, err := os.OpenFile("./temp/info.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
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

	pprof.StartCPUProfile(logFile)
	defer pprof.StopCPUProfile()
	go http.ListenAndServe("0.0.0.0:6060", nil)

	bot, err := tgbotapi.NewBotAPI(os.Getenv("BOT_TOKEN"))
	if err != nil {
		zrlog.Fatal().Msg(err.Error())
		os.Exit(1)
	}

	bot.Debug = true

	zrlog.Info().Msg(fmt.Sprintf("Authorized on account %s", bot.Self.UserName))

	webHookInfo := tgbotapi.NewWebhookWithCert(fmt.Sprintf("https://%s:%s/%s", os.Getenv("BOT_ADDRESS"), os.Getenv("BOT_PORT"), bot.Token), cons.CertPaht)

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

	go http.ListenAndServeTLS("0.0.0.0:8443", cons.CertPaht, cons.KeyPath, nil)

	cacheBotSt = cache.NewCacheBotSt()

	for update := range updates {
		if update.Message == nil && update.CallbackQuery == nil { // ignore non-Message updates and no CallbackQuery
			continue
		}

		if update.Message != nil {
			if update.Message.Photo != nil {

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskPhoto {
					path := *update.Message.Photo

					maxQuality := len(path) - 1

					go getFile(bot, update.Message.Chat.ID, path[maxQuality].FileID, *userPolling, botstate.AskPhoto.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskFile)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.File.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("update.Message.Photo != nil, error sending to user: %+v\n", err))
					}
				} else if cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskFile || cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskFileCorrection {
					path := *update.Message.Photo

					maxQuality := len(path) - 1

					go getFile(bot, update.Message.Chat.ID, path[maxQuality].FileID, *userPolling, botstate.AskFile.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("update.Message.Photo != nil, error sending to user: %+v\n", err))
					}
				}

				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskPhotoCorrection {
					path := *update.Message.Photo

					maxQuality := len(path) - 1

					go getFile(bot, update.Message.Chat.ID, path[maxQuality].FileID, *userPolling, botstate.AskPhoto.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("cacheBotSt.Get(update.Message.Chat.ID) == botstate.ASK_PHOTO_CORRECTION, error sending to user: %+v\n", err))
					}
				}
			}

			if update.Message.Document != nil {
				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskFile || cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskFileCorrection {
					path := *update.Message.Document

					go getFile(bot, update.Message.Chat.ID, path.FileID, *userPolling, botstate.AskFile.EnumIndex())

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("update.Message.Document != nil, error sending to user: %+v\n", err))
					}
				}
			}

			messageByteText := bytes.TrimPrefix([]byte(update.Message.Text), []byte("\xef\xbb\xbf")) // For error deletion of type "invalid character 'ï' looking for beginning of value"
			messageText := string(messageByteText[:])

			switch messageText {

			case botcommand.Start.String():

				err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Здравствуйте, %v!", update.Message.Chat.FirstName), nil, cons.StyleTextCommon, botcommand.Start, "", "", false)

				if err != nil {

					zrlog.Error().Msg(fmt.Sprintf("botcommand.Start.String(), error sending to user %v: %+v", update.Message.Chat.ID, err))

					botBlocked := errs.ErrorHandlerBotBlocked(err)

					if botBlocked {
						go deleteUserPolling(update.Message.Chat.ID, *userPolling)
					}
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.Start)

			case botcommand.GetDiploma.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.GetDiploma)

				err := sentToTelegram(bot, update.Message.Chat.ID, "", nil, cons.StyleTextCommon, botcommand.GetDiploma, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.GetDiploma.String(), error sending to user %v: (%T %+v)\n", update.Message.Chat.ID, err, err))

					botBlocked := errs.ErrorHandlerBotBlocked(err)

					if botBlocked {
						go deleteUserPolling(update.Message.Chat.ID, *userPolling)
					}
				}

			case botcommand.CloseRequisitionStart.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.AskRequisitionNumber)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Номер заявки:", nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CloseRequisitionStart.String(), error sending to admin: %+v\n", err))
				}

			case botcommand.CompleteApplication.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.AskProject)

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите конкурс:", nil, cons.StyleTextCommon, botcommand.CompleteApplication, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CompleteApplication.String(), error sending to user %v: (%T %+v)\n", update.Message.Chat.ID, err, err))
				}

			case botcommand.SelectProject.String():
				if cacheBotSt.Get(update.Message.Chat.ID) == botstate.AskProject {
					if !userPolling.Get(update.Message.Chat.ID).Agree {
						err = sentToTelegram(bot, update.Message.Chat.ID, "Для продолжения необходимо дать согласние на обработку персональных данных. Или нажмите \"Отмена\"", nil, cons.StyleTextCommon, botcommand.WaitingForAcceptance, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botcommand.SelectProject.String(), error sending to user %v: %+v\n", update.Message.Chat.ID, err))
						}
					}
				}

			case botcommand.Cancel.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.Start)

				if thisIsAdmin(update.Message.Chat.ID) {
					go deleteClosingRequisition(update.Message.Chat.ID)
				} else {
					go deleteUserPolling(update.Message.Chat.ID, *userPolling)
				}

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.Cancel.String(), error sending to user: %+v\n", err))
				}

			case botcommand.CancelApplication.String():

				cacheBotSt.Set(update.Message.Chat.ID, botstate.Start)

				if thisIsAdmin(update.Message.Chat.ID) {
					go deleteClosingRequisition(update.Message.Chat.ID)
				} else {
					go deleteUserPolling(update.Message.Chat.ID, *userPolling)
				}

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.CancelApplication.String(), error sending to user: %+v\n", err.Error()))
				}

			case botcommand.Settings.String():

				err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите действие:", nil, cons.StyleTextCommon, botcommand.Settings, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("botcommand.Settings.String(), error sending to user: %+v\n", err))
				}

				cacheBotSt.Set(update.Message.Chat.ID, botstate.Settings)

			default:

				stateBot := cacheBotSt.Get(update.Message.Chat.ID)

				switch stateBot {

				case botstate.GetDiploma:

					err := sentToTelegram(bot, update.Message.Chat.ID, "Если у Вас остались вопросы, пожалуйста, напишите адниминстратору группы:\nhttps://vk.com/topic-138597952_49394008", nil, cons.StyleTextHTML, botcommand.Undefined, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, error sending to user: (%T %+v)\n", err, err))
					}

				case botstate.AskPublicationDate: //This botstate is only available for the admin

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PublicationDate, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskPublicationLink)

					err := sentToTelegram(bot, update.Message.Chat.ID, "Укажите ссылку на опубликованную работу:", nil, cons.StyleTextCommon, botcommand.GetPublicationLink, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskPublicationDate, error sending to user: (%T %+v)\n", err, err))
					}

				case botstate.AskPublicationLink: //This botstate is only available for the admin

					closingRequisition.Set(update.Message.Chat.ID, enumapplic.PublicationLink, messageText)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskPublicationLink, error sending to user: %+v\n", err))
					}

				case botstate.AskRequisitionNumber: // This botstate is available only for the admin

					_, err := strconv.Atoi(messageText)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskRequisitionNumber, error convert strconv.Atoi: %+v\n", err))

						err := sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Некорректно введен номер заявки.\n Пожалуста, введите цифрами:", nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskRequisitionNumber, error sending to user: %+v\n", err))
						}
					} else {
						closingRequisition.Set(update.Message.Chat.ID, enumapplic.RequisitionNumber, messageText)
						closingRequisition.Set(update.Message.Chat.ID, enumapplic.TableDB, cons.TableDB)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.AskDegree)

						err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите степень:", nil, cons.StyleTextCommon, botcommand.SelectDegree, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskRequisitionNumber, error sending to admin: %+v\n", err))
						}
					}

				case botstate.AskFNP:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskAge)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника/группы участников.\n(Если не хотите указывать возраст - нажмите \"Далее\").", enumapplic.Age.EnumIndex()), nil, cons.StyleTextCommon, botcommand.AskAge, "", "", false)

					if err != nil {
						zrlog.Error().Msg(err.Error())
					}

				case botstate.AskFNPCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNP, messageText, 0)

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskFNPCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskAge:

					if messageText == botcommand.Down.String() {
						userPolling.Set(update.Message.Chat.ID, enumapplic.Age, cons.Zero, 0)
					} else {
						userPolling.Set(update.Message.Chat.ID, enumapplic.Age, messageText, 0)
					}

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskNameInstitution)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NameInstitution.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskAge, error sending to user: %+v\n", err))
					}

				case botstate.AskAgeCorrection:

					if messageText == botcommand.Down.String() {
						userPolling.Set(update.Message.Chat.ID, enumapplic.Age, cons.Zero, 0)
					} else {
						userPolling.Set(update.Message.Chat.ID, enumapplic.Age, messageText, 0)
					}

					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskAgeCorrection, error sending to user: %+v\n", err.Error()))
					}

				case botstate.AskNameInstitution:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NameInstitution, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskLocality)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.Locality.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskNameInstitution, error sending to user: %+v\n", err))
					}

				case botstate.AskNameInstitutionCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NameInstitution, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskNameInstitutionCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskLocality:

					userPolling.Set(update.Message.Chat.ID, enumapplic.Locality, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskNamingUnit)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NamingUnit.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskLocality, error sending to user: %+v\n", err))
					}

				case botstate.AskLocalityCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.Locality, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskLocalityCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskNamingUnit:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NamingUnit, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskPublicationTitle)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PublicationTitle.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskNamingUnit, error sending to user: %+v\n", err))
					}

				case botstate.AskNamingUnitCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.NamingUnit, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskNamingUnitCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskPublicationTitle:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PublicationTitle, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskFNPLeader)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (через запятую, если двое) или нажмите \"Далее\" если нет руководителя:", enumapplic.FNPLeader.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectFNPLeader, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskPublicationTitle, error sending to user: %+v\n", err))
					}

				case botstate.AskPublicationTitleCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.PublicationTitle, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskPublicationTitleCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskFNPLeader:

					if messageText != botcommand.Down.String() {
						userPolling.Set(update.Message.Chat.ID, enumapplic.FNPLeader, messageText, 0)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.AskEmail)

						err := sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты 📧:", enumapplic.Email.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskFNPLeader, error sending to user: %+v\n", err))
						}
					} else {
						userPolling.Set(update.Message.Chat.ID, enumapplic.FNPLeader, "", 0)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.AskEmail)

						err := sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты 📧:", enumapplic.Email.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskFNPLeader, error sending to user: %+v\n", err))
						}
					}

				case botstate.AskFNPLeaderCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.FNPLeader, messageText, 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskFNPLeaderCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskEmail:

					if validEmail(strings.TrimSpace(messageText)) {
						userPolling.Set(update.Message.Chat.ID, enumapplic.Email, strings.TrimSpace(messageText), 0)
						cacheBotSt.Set(update.Message.Chat.ID, botstate.AskDocumentType)

						err := sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DocumentType.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectDocumentType, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskEmail, error sending to user: %+v\n", err))
						}

					} else {
						err := sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Некорректный адрес электронной почты.\n Пожалуйста, введите адрес email правильно.", nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskEmail, error sending to user: %+v\n", err))
						}
					}

				case botstate.AskEmailCorrection:

					userPolling.Set(update.Message.Chat.ID, enumapplic.Email, strings.TrimSpace(messageText), 0)
					cacheBotSt.Set(update.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.AskEmailCorrection, error sending to user: %+v\n", err))
					}

				case botstate.AskCheckData:

					if messageText == botcommand.SelectCorrection.String() {
						cacheBotSt.Set(update.Message.Chat.ID, botstate.SelectCorrection)

						err = sentToTelegram(bot, update.Message.Chat.ID, "Выберите пункт который нужно исправить:", nil, cons.StyleTextCommon, botcommand.SelectCorrection, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData: %+v\n", err))
						}

					} else if messageText == botcommand.Confirm.String() {

						if !thisIsAdmin(update.Message.Chat.ID) {
							err := sentToTelegram(bot, update.Message.Chat.ID, "Регистрирую...", nil, cons.StyleTextCommon, botcommand.RecordToDB, "", "", false)
							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, sending for user: %+v\n", err))
							}

							ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, unable to establish connection to database for users: %+v\n", err.Error()))
							}
							defer dbpool.Close()

							dbpool.Config().MaxConns = 7

							err = AddRequisition(ctx, update.Message.Chat.ID, dbpool)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, error append requisition to db for user: %+v\n", err.Error()))
							}

							ok, err := ConvertRequisitionToPDF(update.Message.Chat.ID)

							if err != nil || !ok {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, error converting requisition into PDF for user: %+v\n", err.Error()))
							} else {
								numReq := userPolling.Get(update.Message.Chat.ID).RequisitionNumber
								pathReqPDF := fmt.Sprintf("./external/files/usersfiles/Заявка_№%v.pdf", numReq)

								userPolling.Set(update.Message.Chat.ID, enumapplic.RequisitionPDF, pathReqPDF, 0)

								err = sentToTelegramPDF(bot, update.Message.Chat.ID, pathReqPDF, "", botcommand.Undefined)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, error sending file pdf to user: %v\n", err))
								}

								//Email
								t := time.Now()
								formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

								send, err := SentEmail(os.Getenv("ADMIN_EMAIL"), update.Message.Chat.ID, true, true, fmt.Sprintf("Заявка №%v от %s (%s)", numReq, formattedTime, userPolling.Get(update.Message.Chat.ID).DocumentType), userPolling.Get(update.Message.Chat.ID).Files, "")
								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, error sending letter to admin's email: %+v\n", err.Error()))
								}

								if send {
									cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)
									go deleteUserPolling(update.Message.Chat.ID, *userPolling)
								}

								err = sentToTelegram(bot, update.Message.Chat.ID, "Поздравляем, Ваша заявка зарегестрирована! Благодарим Вас за участие, ваша заявка будет обработана в течение трех дней.", nil, cons.StyleTextCommon, botcommand.RecordToDB, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								}
							}
						}

						if thisIsAdmin(update.Message.Chat.ID) {
							ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
							defer cancel()

							dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
							dbpool.Config().MaxConns = 12

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, unable to establish connection to database for admin: %+v\n", err.Error()))
							}
							defer dbpool.Close()

							err = GetRequisitionForAdmin(ctx, update.Message.Chat.ID, closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber, closingRequisition.Get(update.Message.Chat.ID).UserData.Degree, closingRequisition.Get(update.Message.Chat.ID).UserData.TableDB, closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationDate, closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationLink, dbpool)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, GetRequisitionForAdmin(): %+v\n", err.Error()))

								err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Ошибка закрытия заявки!\n%s", err.Error()), nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, sending for admin: %+v\n", err.Error()))
								}

							} else {

								wg.Add(2)
								go FillInCertificatesPDFForms(&wg, update.Message.Chat.ID)
								go FillInDiplomasPDFForms(&wg, update.Message.Chat.ID)
								wg.Wait()

								// Send to admin for check
								temp := ""
								for _, path := range closingRequisition.Get(update.Message.Chat.ID).UserData.Files {
									// When some files are the same
									if temp != "" && temp == path {
										continue
									}

									err = sentToTelegramPDF(bot, update.Message.Chat.ID, path, "", botcommand.Undefined)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, send to admin for check, error sending file pdf to admin: %v\n", err))
									}

									temp = path
								}

								err = sentToTelegram(bot, update.Message.Chat.ID, "Подтвердить или отменить закрытие?", nil, cons.StyleTextCommon, botcommand.CheckPDFFiles, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("botstate.AskCheckData, sending for admin: %+v\n", err.Error()))
								}
							}
						}
					} else if messageText == botcommand.SendPDFFiles.String() && thisIsAdmin(update.Message.Chat.ID) {
						ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
						defer cancel()

						dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
						dbpool.Config().MaxConns = 12

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf(" else if messageText == botcommand.SendPDFFiles.String() && thisIsAdmin(update.Message.Chat.ID), Unable to establish connection to database: %+v\n", err.Error()))
						}
						defer dbpool.Close()

						switch closingRequisition.Get(update.Message.Chat.ID).UserData.PlaceDeliveryDocuments {

						case cons.PlaceDeliveryOfDocuments1: // The user wants to get certificate/diploma on Email

							t := time.Now()
							formattedTime := fmt.Sprintf("%02d.%02d.%d", t.Day(), t.Month(), t.Year())

							sent, err := SentEmail(closingRequisition.Get(update.Message.Chat.ID).UserData.Email, closingRequisition.Get(update.Message.Chat.ID).UserID, false, false, fmt.Sprintf("%s №%v от %s ", closingRequisition.Get(update.Message.Chat.ID).UserData.DocumentType, closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber, formattedTime), closingRequisition.Get(update.Message.Chat.ID).UserData.Files, "")
							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments1:, error sending letter to users's email: %+v\n", err.Error()))
							}

							if sent {
								err = UpdateRequisition(ctx, true, true, closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber, closingRequisition.Get(update.Message.Chat.ID).UserData.TableDB, closingRequisition.Get(update.Message.Chat.ID).UserData.Degree, closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationLink, closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationDate, dbpool)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments1:, UpdateRequisition() for admin: %v\n", err))

									err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Ошибка! Заявка №%v НЕ закрыта!", closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments1: %v\n", err))
									}

								} else {
									err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Заявка №%v закрыта!", closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RecordToDB, "", "", false)

									if err != nil {
										zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments1: %+v\n", err.Error()))
									}

									cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)
									go deleteClosingRequisition(update.Message.Chat.ID)
								}
							}

							if !sent {
								err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Не удалось отправить письмо. Заявка №%v не закрыта.", closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

								if err != nil {
									zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
								}

								cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)
								go deleteClosingRequisition(update.Message.Chat.ID)
							}

						case cons.PlaceDeliveryOfDocuments2: // The user wants to get certificate/diploma in Telegram

							err = UpdateRequisition(ctx, true, false, closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber,
								closingRequisition.Get(update.Message.Chat.ID).UserData.TableDB, closingRequisition.Get(update.Message.Chat.ID).UserData.Degree,
								closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationLink,
								closingRequisition.Get(update.Message.Chat.ID).UserData.PublicationDate, dbpool)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments2, UpdateRequisition() for admin: %+v\n", err))
							}

							err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("Заявка №%v закрыта!", closingRequisition.Get(update.Message.Chat.ID).UserData.RequisitionNumber), nil, cons.StyleTextCommon, botcommand.RecordToDB, "", "", false)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("case cons.PlaceDeliveryOfDocuments2: %+v\n", err.Error()))
							}

							cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)
							go deleteClosingRequisition(update.Message.Chat.ID)

						}
					} else if messageText == botcommand.CancelApplication.String() {
						cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)

						if thisIsAdmin(update.Message.Chat.ID) {
							go deleteClosingRequisition(update.Message.Chat.ID)
						}

						if !thisIsAdmin(update.Message.Chat.ID) {
							go deleteUserPolling(update.Message.Chat.ID, *userPolling)
						}

					} else if messageText == botcommand.CancelCloseRequisition.String() && thisIsAdmin(update.Message.Chat.ID) { // excess condition
						err = sentToTelegram(bot, update.Message.Chat.ID, "Выход в главное меню", nil, cons.StyleTextCommon, botcommand.Cancel, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
						}

						cacheBotSt.Set(update.Message.Chat.ID, botstate.Undefined)
						go deleteClosingRequisition(update.Message.Chat.ID)
					}

				case botstate.Undefined:

					err := sentToTelegram(bot, update.Message.Chat.ID, "Если у Вас есть вопросы, пожалуйста, напишите адниминстратору группы:\nhttps://vk.com/topic-138597952_49394008", nil, cons.StyleTextHTML, botcommand.Undefined, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.Undefined, error sending to user: (%T %+v)\n", err, err))
					}

				}
			}
		}

		if update.CallbackQuery != nil {
			callbackQueryData := bytes.TrimPrefix([]byte(update.CallbackQuery.Data), []byte("\xef\xbb\xbf")) // For error deletion of type "invalid character 'ï' looking for beginning of value"
			callbackQueryText := string(callbackQueryData[:])

			var description string

			switch callbackQueryText {

			case string(cons.ContestTitmouse): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestTitmouse.String(), 0)

				description = GetConciseDescription(string(cons.ContestTitmouse))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, 	case string(cons.ContestTitmouse): %+v\n", err.Error()))
				}

			case string(cons.ContestDefendersFatherland): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestDefendersFatherland.String(), 0)

				description = GetConciseDescription(string(cons.ContestDefendersFatherland))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestDefendersFatherland): %+v\n", err.Error()))
				}

			case string(cons.ContestMather): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestMather.String(), 0)

				description = GetConciseDescription(string(cons.ContestMather))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestMather): %+v\n", err.Error()))
				}

			case string(cons.ContestFather): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestFather.String(), 0)

				description = GetConciseDescription(string(cons.ContestFather))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestFather): %+v\n", err.Error()))
				}

			case string(cons.ContestMyFamily): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestMyFamily.String(), 0)

				description = GetConciseDescription(string(cons.ContestMyFamily))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestMyFamily): %+v\n", err.Error()))
				}

			case string(cons.ContestChildProtectionDay): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestChildProtectionDay.String(), 0)

				description = GetConciseDescription(string(cons.ContestChildProtectionDay))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestChildProtectionDay): %+v\n", err.Error()))
				}

			case string(cons.ContestMotherRussia): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestMotherRussia.String(), 0)

				description = GetConciseDescription(string(cons.ContestMotherRussia))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestMotherRussia): %+v\n", err.Error()))
				}

			case string(cons.ContestAutumn): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestAutumn.String(), 0)

				description = GetConciseDescription(string(cons.ContestAutumn))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestAutumn): %+v\n", err.Error()))
				}

			case string(cons.ContestWinter): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestWinter.String(), 0)

				description = GetConciseDescription(string(cons.ContestWinter))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestWinter): %+v\n", err.Error()))
				}

			case string(cons.ContestSnowflakes): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSnowflakes.String(), 0)

				description = GetConciseDescription(string(cons.ContestSnowflakes))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSnowflakes): %+v\n", err.Error()))
				}

			case string(cons.ContestSnowman): //CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSnowman.String(), 0)

				description = GetConciseDescription(string(cons.ContestSnowman))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSnowman): %+v\n", err.Error()))
				}

			case string(cons.ContestSymbol): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSymbol.String(), 0)

				description = GetConciseDescription(string(cons.ContestSymbol))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSymbol): %+v\n", err.Error()))
				}

			case string(cons.ContestHearts): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestHearts.String(), 0)

				description = GetConciseDescription(string(cons.ContestHearts))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestHeart): %+v\n", err.Error()))
				}

			case string(cons.ContestSecrets): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSecrets.String(), 0)

				description = GetConciseDescription(string(cons.ContestSecrets))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSecrets): %+v\n", err.Error()))
				}

			case string(cons.ContestBirdsFeeding): // CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestBirdsFeeding.String(), 0)

				description = GetConciseDescription(string(cons.ContestBirdsFeeding))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestBirdsFeeding): %+v\n", err.Error()))
				}

			case string(cons.ContestShrovetide): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestShrovetide.String(), 0)

				description = GetConciseDescription(string(cons.ContestShrovetide))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestShrovetide): %+v\n", err.Error()))
				}

			case string(cons.ContestFable): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestFable.String(), 0)

				description = GetConciseDescription(string(cons.ContestFable))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestFable): %+v\n", err.Error()))
				}

			case string(cons.ContestSpring): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSpring.String(), 0)

				description = GetConciseDescription(string(cons.ContestSpring))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSpring): %+v\n", err.Error()))
				}

			case string(cons.ContestMarchEighth): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestMarchEighth.String(), 0)

				description = GetConciseDescription(string(cons.ContestMarchEighth))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestMarchEighth): %+v\n", err.Error()))
				}

			case string(cons.ContestEarth): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestEarth.String(), 0)

				description = GetConciseDescription(string(cons.ContestEarth))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestEarth): %+v\n", err.Error()))
				}

			case string(cons.ContestSpaceAdventures): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSpaceAdventures.String(), 0)

				description = GetConciseDescription(string(cons.ContestSpaceAdventures))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSpaceAdventures): %+v\n", err.Error()))
				}

			case string(cons.ContestFeatheredFriends): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestFeatheredFriends.String(), 0)

				description = GetConciseDescription(string(cons.ContestFeatheredFriends))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestFeatheredFriends): %+v\n", err.Error()))
				}

			case string(cons.ContestTheatricalBackstage): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestTheatricalBackstage.String(), 0)

				description = GetConciseDescription(string(cons.ContestTheatricalBackstage))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestTheatricalBackstage): %+v\n", err.Error()))
				}
			case string(cons.ContestOurFriends): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestOurFriends.String(), 0)

				description = GetConciseDescription(string(cons.ContestOurFriends))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestOurFriends): %+v\n", err.Error()))
				}

			case string(cons.ContestVictoryDay): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestVictoryDay.String(), 0)

				description = GetConciseDescription(string(cons.ContestVictoryDay))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestVictoryDay): %+v\n", err.Error()))
				}

			case string(cons.ContestPrimroses): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestPrimroses.String(), 0)

				description = GetConciseDescription(string(cons.ContestPrimroses))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestPrimroses): %+v\n", err.Error()))
				}

			case string(cons.ContestFire): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestFire.String(), 0)

				description = GetConciseDescription(string(cons.ContestFire))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestFire): %+v\n", err.Error()))
				}

			case string(cons.ContestTrafficLight): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestTrafficLight.String(), 0)

				description = GetConciseDescription(string(cons.ContestTrafficLight))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestTrafficLight): %+v\n", err.Error()))
				}

			case string(cons.ContestSummerPalette): // CallBackQwery
				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Contest, cons.ContestSummerPalette.String(), 0)

				description = GetConciseDescription(string(cons.ContestSummerPalette))

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, description, nil, cons.StyleTextHTML, botcommand.SelectProject, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.ContestSummerPalette): %+v\n", err.Error()))
				}

			case string(cons.Degree1): // CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskDegree {
					closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Degree, cons.Degree1.String())
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPublicationDate)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GetPublicationDate, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.Degree1): %+v\n", err.Error()))
					}
				}

			case string(cons.Degree2): // CallBackQwery

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {
					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskDegree {
						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Degree, cons.Degree2.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPublicationDate)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GetPublicationDate, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.Degree2): %+v\n", err.Error()))
						}
					}
				}

			case string(cons.Degree3): // CallBackQwery

				if thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {
					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskDegree {
						closingRequisition.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Degree, cons.Degree3.String())
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPublicationDate)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Укажите дату публикации работы:", nil, cons.StyleTextCommon, botcommand.GetPublicationDate, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case string(cons.Degree3): %+v\n", err.Error()))
						}
					}
				}

			case cons.Certificate.String(): // CallBackQwery

				if !thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DocumentType, string(cons.Certificate), 0)
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Diploma, "false", 0)
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.TableDB, 0)

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskDocumentTypeCorrection {
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskCheckData)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.Certificate.String(): %+v\n", err.Error()))
						}
					} else {
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPlaceDeliveryOfDocuments)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PlaceDeliveryOfDocuments.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectPlaceDeliveryOfDocuments, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.Certificate.String(): %+v\n", err.Error()))
						}
					}
				}

			case cons.CertificateAndDiploma.String(): // CallBackQwery

				if !thisIsAdmin(update.CallbackQuery.Message.Chat.ID) {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.DocumentType, string(cons.Diploma), 0)
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Diploma, "true", 0)
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.TableDB, cons.TableDB, 0)

					if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskDocumentTypeCorrection {
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskCheckData)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CertificateAndDiploma.String(): %+v\n", err.Error()))
						}
					} else {
						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPlaceDeliveryOfDocuments)

						err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PlaceDeliveryOfDocuments.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectPlaceDeliveryOfDocuments, "", "", false)

						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.CertificateAndDiploma.String(): %+v\n", err.Error()))
						}
					}
				}

			case cons.PlaceDeliveryOfDocuments1: // CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskPlaceDeliveryOfDocuments {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PlaceDeliveryOfDocuments, cons.PlaceDeliveryOfDocuments1, 0)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPhoto)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фотографию (только одну) Вашей работы:", enumapplic.Photo.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.PlaceDeliveryOfDocuments1: %+v\n", err.Error()))
					}
				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskPlaceDeliveryOfDocumentsCorrection {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PlaceDeliveryOfDocuments, cons.PlaceDeliveryOfDocuments1, 0)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, cons.PlaceDeliveryOfDocuments1: %+v\n", err.Error()))
					}
				}

			case cons.PlaceDeliveryOfDocuments2: // CallBackQwery

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskPlaceDeliveryOfDocuments {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PlaceDeliveryOfDocuments, cons.PlaceDeliveryOfDocuments2, 0)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPhoto)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фотографию (только одну) Вашей работы:", enumapplic.Photo.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.PlaceDeliveryOfDocuments2: %+v\n", err.Error()))
					}
				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskPlaceDeliveryOfDocumentsCorrection {
					userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.PlaceDeliveryOfDocuments, cons.PlaceDeliveryOfDocuments2, 0)

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case cons.PlaceDeliveryOfDocuments2: %+v\n", err.Error()))
					}
				}

			case enumapplic.CancelCorrection.String(): // CallBackQwery "CancelCorrection"
				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskCheckData)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "⚠️ Пожалуйста, проверьте введенные данные:", nil, cons.StyleTextCommon, botcommand.CheckData, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.CancelCorrection.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.FNP.String(): // CallBackQwery "FNP"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskFNPCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО участника или группу участников (например, \"страшая группа №7\" или \"старшая группа \"Карамельки\"):", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FNP.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.Age.String(): // CallBackQwery "Age"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskAgeCorrection)

					err = sentToTelegram(bot, update.Message.Chat.ID, fmt.Sprintf("%v. Введите возраст участника/группы участников.\nЕсли не хотите указывать возраст - нажмите \"Далее\".", enumapplic.Age.EnumIndex()), nil, cons.StyleTextCommon, botcommand.AskAge, "", "", false)

					if err != nil {
						zrlog.Error().Msg(err.Error())
					}
				}

			case enumapplic.NameInstitution.String(): // CallBackQwery "NameInstitution"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskNameInstitutionCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название учреждения (сокращенное):", enumapplic.NameInstitution.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.NameInstitution.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.Locality.String(): // CallBackQwery "Locality"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskLocalityCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите населенный пункт:", enumapplic.Locality.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.Locality.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.NamingUnit.String(): // CallBackQwery "NamingUnit"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskNamingUnitCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите номинацию:", enumapplic.NamingUnit.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.NamingUnit.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.PublicationTitle.String(): // CallBackQwery "PublicationTitle"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPublicationTitleCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите название работы:", enumapplic.PublicationTitle.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.PublicationTitle.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.FNPLeader.String(): // CallBackQwery "FNPLeader"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskFNPLeaderCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО руководителя (через запятую, если два):", enumapplic.FNPLeader.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FNPLeader.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.Email.String(): // CallBackQwery "Email"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskEmailCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите адрес электронной почты 📧:", enumapplic.Email.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.Enail.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.DocumentType.String(): // CallBackQwery "DocumentType"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskDocumentTypeCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите тип документа:", enumapplic.DocumentType.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectDocumentType, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.DocumentType.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.PlaceDeliveryOfDocuments.String(): // CallBackQwery "PlaceDeliveryOfDocuments"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPlaceDeliveryOfDocumentsCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Выберите место получения документа:", enumapplic.PlaceDeliveryOfDocuments.EnumIndex()), nil, cons.StyleTextCommon, botcommand.SelectPlaceDeliveryOfDocuments, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.PlaceDeliveryOfDocuments.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.Photo.String(): // CallBackQwery "PHOTO"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskPhotoCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Отправьте фотографию (только одну) Вашей работы:", enumapplic.Photo.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.Photo.String(): %+v\n", err.Error()))
					}
				}

			case enumapplic.File.String(): // CallBackQwery "FILE"

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.SelectCorrection {
					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskFileCorrection)

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Прикрепите квитанцию об оплате:", enumapplic.File.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("CallBackQwery, case enumapplic.FILE.String(): %+v\n", err.Error()))
					}
				}

			case cons.Agree.String(): //CallBackQwery

				userPolling.Set(update.CallbackQuery.Message.Chat.ID, enumapplic.Agree, "", 0)

				err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Согласие на обработку персональных данных получено", nil, cons.StyleTextCommon, botcommand.WaitingForAcceptance, "", "", false)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("Error sending to user: %+v\n", err.Error()))
				}

				if cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID) == botstate.AskProject {

					err = sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("%v. Введите ФИО участника или группу участников (например, \"страшая группа №7\" или \"старшая группа \"Карамельки\"):", enumapplic.FNP.EnumIndex()), nil, cons.StyleTextCommon, botcommand.ContinueDataPolling, "", "", false)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botcommand.SelectProject.String(), error sending to user: %+v\n", err))
					}

					cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.AskFNP)
				}

			default: //CallBackQwery

				stateBot := cacheBotSt.Get(update.CallbackQuery.Message.Chat.ID)

				switch stateBot {

				case botstate.GetDiploma: //This command is only available to users

					requisitionNumber, _ := strconv.Atoi(callbackQueryText)

					ctx, cancel := context.WithTimeout(context.Background(), 17*time.Second)
					defer cancel()

					dbpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
					defer dbpool.Close()
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, unable to establish connection to database: %+v\n", err.Error()))
					}

					dbpool.Config().MaxConns = 2

					er := GetRequisitionForUser(ctx, update.CallbackQuery.Message.Chat.ID, int64(requisitionNumber), dbpool)
					if er != nil {
						zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, GetRequisitionForUser(): %+v\n", er))
					}

					switch {

					case strings.TrimSpace(userPolling.Get(update.CallbackQuery.Message.Chat.ID).PublicationLink) == "":

						err := sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, fmt.Sprintf("Ваша заявка № %v от %v находится в работе.", requisitionNumber, unixNanoToDateString(userPolling.Get(update.CallbackQuery.Message.Chat.ID).RequisitionStartDate)), nil, cons.StyleTextCommon, botcommand.Start, "", "", false)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, error sending to user: %+v\n", err))
						}

					default:

						err := sentToTelegram(bot, update.CallbackQuery.Message.Chat.ID, "Отправляю...", nil, cons.StyleTextCommon, botcommand.Start, "", "", false)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, error sending to user: %+v\n", err))
						}

						wg := &sync.WaitGroup{}

						wg.Add(2)
						go FillInCertificatesPDFForms(wg, update.CallbackQuery.Message.Chat.ID)
						go FillInDiplomasPDFForms(wg, update.CallbackQuery.Message.Chat.ID)
						wg.Wait()

						temp := ""
						for _, path := range userPolling.Get(update.CallbackQuery.Message.Chat.ID).Files {
							// When some files are the same
							if temp != "" && temp == path {
								continue
							}

							err := sentToTelegramPDF(bot, update.CallbackQuery.Message.Chat.ID, path, "", botcommand.Undefined)

							if err != nil {
								zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, error sending file pdf to user: %v\n", err))
							}

							temp = path
						}

						er := UpdateRequisition(ctx, false, false, userPolling.Get(update.CallbackQuery.Message.Chat.ID).RequisitionNumber, userPolling.Get(update.CallbackQuery.Message.Chat.ID).TableDB, 0, "", "", dbpool)

						if er != nil {
							zrlog.Error().Msg(fmt.Sprintf("botstate.GetDiploma, UpdateRequisition(): %+v\n", er))
						}

						cacheBotSt.Set(update.CallbackQuery.Message.Chat.ID, botstate.Undefined)

						go deleteUserPolling(update.CallbackQuery.Message.Chat.ID, *userPolling)

					}
				}

			}
		}
	}
}

func sentToTelegram(bot *tgbotapi.BotAPI, id int64, message string, lenBody map[int]int, styleText string, command botcommand.BotCommand, button, header string, PDF bool) error {

	defer func() error {
		if errr := recover(); errr != nil {
			//zrlog.Error().Msg(fmt.Sprintf("recover: %+v\n", errr))
			return fmt.Errorf("recover: %+v", errr)
		}

		return nil
	}()

	switch command {

	case botcommand.SelectCorrection:

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

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.Age.EnumIndex(), enumapplic.Age.String()), enumapplic.Age.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.NameInstitution.EnumIndex(), enumapplic.NameInstitution.String()), enumapplic.NameInstitution.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.Locality.EnumIndex(), enumapplic.Locality.String()), enumapplic.Locality.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton4)

			inlineKeyboardButton5 = append(inlineKeyboardButton5, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.NamingUnit.EnumIndex(), enumapplic.NamingUnit.String()), enumapplic.NamingUnit.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton5)

			inlineKeyboardButton6 = append(inlineKeyboardButton6, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.PublicationTitle.EnumIndex(), enumapplic.PublicationTitle.String()), enumapplic.PublicationTitle.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton6)

			inlineKeyboardButton7 = append(inlineKeyboardButton7, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()), enumapplic.FNPLeader.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton7)

			inlineKeyboardButton8 = append(inlineKeyboardButton8, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.Email.EnumIndex(), enumapplic.Email.String()), enumapplic.Email.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton8)

			inlineKeyboardButton9 = append(inlineKeyboardButton9, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.DocumentType.EnumIndex(), enumapplic.DocumentType.String()), enumapplic.DocumentType.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton9)

			inlineKeyboardButton10 = append(inlineKeyboardButton10, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.PlaceDeliveryOfDocuments.EnumIndex(), enumapplic.PlaceDeliveryOfDocuments.String()), enumapplic.PlaceDeliveryOfDocuments.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton10)

			inlineKeyboardButton11 = append(inlineKeyboardButton11, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.Photo.EnumIndex(), enumapplic.Photo.String()), enumapplic.Photo.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton11)

			inlineKeyboardButton12 = append(inlineKeyboardButton12, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%v. %s", enumapplic.File.EnumIndex(), enumapplic.File.String()), enumapplic.File.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton12)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SelectCorrection: %w", err)
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
			inlineKeyboardButton13 = append(inlineKeyboardButton13, tgbotapi.NewInlineKeyboardButtonData(enumapplic.CancelCorrection.String(), enumapplic.CancelCorrection.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton13)
			inlineKeyboardMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SELECT_CORRECTIONT: %w", err)
			}
		}

	case botcommand.Start:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.Start: (%T, %+v), %w", err, err, err)
		}

	case botcommand.AccessDenied:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardMainMenue

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.AccessDenied: %w", err)
		}

		deleteUserPolling(id, *userPolling)

	case botcommand.Cancel:

		msg := tgbotapi.NewMessage(id, message, styleText) // enter to main menue

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.Cancel: %w", err)
		}

		cacheBotSt.Set(id, botstate.Start)

		if thisIsAdmin(id) {
			deleteClosingRequisition(id)
		} else {
			deleteUserPolling(id, *userPolling)
		}

	case botcommand.CompleteApplication:

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
			inlineKeyboardButton15 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton16 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton17 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton18 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton19 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton20 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton21 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton22 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton23 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton24 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton25 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton26 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton27 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton28 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton29 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.ContestTitmouse.String(), string(cons.ContestTitmouse)))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.ContestMather.String(), string(cons.ContestMather)))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.ContestFather.String(), string(cons.ContestFather)))
			rowsButton = append(rowsButton, inlineKeyboardButton3)

			inlineKeyboardButton4 = append(inlineKeyboardButton4, tgbotapi.NewInlineKeyboardButtonData(cons.ContestMyFamily.String(), string(cons.ContestMyFamily)))
			rowsButton = append(rowsButton, inlineKeyboardButton4)

			inlineKeyboardButton5 = append(inlineKeyboardButton5, tgbotapi.NewInlineKeyboardButtonData(cons.ContestChildProtectionDay.String(), string(cons.ContestChildProtectionDay)))
			rowsButton = append(rowsButton, inlineKeyboardButton5)

			inlineKeyboardButton6 = append(inlineKeyboardButton6, tgbotapi.NewInlineKeyboardButtonData(cons.ContestMotherRussia.String(), string(cons.ContestMotherRussia)))
			rowsButton = append(rowsButton, inlineKeyboardButton6)

			inlineKeyboardButton7 = append(inlineKeyboardButton7, tgbotapi.NewInlineKeyboardButtonData(cons.ContestAutumn.String(), string(cons.ContestAutumn)))
			rowsButton = append(rowsButton, inlineKeyboardButton7)

			inlineKeyboardButton8 = append(inlineKeyboardButton8, tgbotapi.NewInlineKeyboardButtonData(cons.ContestWinter.String(), string(cons.ContestWinter)))
			rowsButton = append(rowsButton, inlineKeyboardButton8)

			inlineKeyboardButton9 = append(inlineKeyboardButton9, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSnowflakes.String(), string(cons.ContestSnowflakes)))
			rowsButton = append(rowsButton, inlineKeyboardButton9)

			inlineKeyboardButton10 = append(inlineKeyboardButton10, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSnowman.String(), string(cons.ContestSnowman)))
			rowsButton = append(rowsButton, inlineKeyboardButton10)

			inlineKeyboardButton11 = append(inlineKeyboardButton11, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSymbol.String(), string(cons.ContestSymbol)))
			rowsButton = append(rowsButton, inlineKeyboardButton11)

			inlineKeyboardButton12 = append(inlineKeyboardButton12, tgbotapi.NewInlineKeyboardButtonData(cons.ContestHearts.String(), string(cons.ContestHearts)))
			rowsButton = append(rowsButton, inlineKeyboardButton12)

			inlineKeyboardButton13 = append(inlineKeyboardButton13, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSecrets.String(), string(cons.ContestSecrets)))
			rowsButton = append(rowsButton, inlineKeyboardButton13)

			inlineKeyboardButton14 = append(inlineKeyboardButton14, tgbotapi.NewInlineKeyboardButtonData(cons.ContestBirdsFeeding.String(), string(cons.ContestBirdsFeeding)))
			rowsButton = append(rowsButton, inlineKeyboardButton14)

			inlineKeyboardButton15 = append(inlineKeyboardButton15, tgbotapi.NewInlineKeyboardButtonData(cons.ContestShrovetide.String(), string(cons.ContestShrovetide)))
			rowsButton = append(rowsButton, inlineKeyboardButton15)

			inlineKeyboardButton16 = append(inlineKeyboardButton16, tgbotapi.NewInlineKeyboardButtonData(cons.ContestFable.String(), string(cons.ContestFable)))
			rowsButton = append(rowsButton, inlineKeyboardButton16)

			inlineKeyboardButton17 = append(inlineKeyboardButton17, tgbotapi.NewInlineKeyboardButtonData(cons.ContestDefendersFatherland.String(), string(cons.ContestDefendersFatherland)))
			rowsButton = append(rowsButton, inlineKeyboardButton17)

			inlineKeyboardButton18 = append(inlineKeyboardButton18, tgbotapi.NewInlineKeyboardButtonData(cons.ContestVictoryDay.String(), string(cons.ContestVictoryDay)))
			rowsButton = append(rowsButton, inlineKeyboardButton18)

			inlineKeyboardButton19 = append(inlineKeyboardButton19, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSpring.String(), string(cons.ContestSpring)))
			rowsButton = append(rowsButton, inlineKeyboardButton19)

			inlineKeyboardButton20 = append(inlineKeyboardButton20, tgbotapi.NewInlineKeyboardButtonData(cons.ContestPrimroses.String(), string(cons.ContestPrimroses)))
			rowsButton = append(rowsButton, inlineKeyboardButton20)

			inlineKeyboardButton21 = append(inlineKeyboardButton21, tgbotapi.NewInlineKeyboardButtonData(cons.ContestMarchEighth.String(), string(cons.ContestMarchEighth)))
			rowsButton = append(rowsButton, inlineKeyboardButton21)

			inlineKeyboardButton22 = append(inlineKeyboardButton22, tgbotapi.NewInlineKeyboardButtonData(cons.ContestEarth.String(), string(cons.ContestEarth)))
			rowsButton = append(rowsButton, inlineKeyboardButton22)

			inlineKeyboardButton23 = append(inlineKeyboardButton23, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSpaceAdventures.String(), string(cons.ContestSpaceAdventures)))
			rowsButton = append(rowsButton, inlineKeyboardButton23)

			inlineKeyboardButton24 = append(inlineKeyboardButton24, tgbotapi.NewInlineKeyboardButtonData(cons.ContestFeatheredFriends.String(), string(cons.ContestFeatheredFriends)))
			rowsButton = append(rowsButton, inlineKeyboardButton24)

			inlineKeyboardButton25 = append(inlineKeyboardButton25, tgbotapi.NewInlineKeyboardButtonData(cons.ContestTheatricalBackstage.String(), string(cons.ContestTheatricalBackstage)))
			rowsButton = append(rowsButton, inlineKeyboardButton25)

			inlineKeyboardButton26 = append(inlineKeyboardButton26, tgbotapi.NewInlineKeyboardButtonData(cons.ContestOurFriends.String(), string(cons.ContestOurFriends)))
			rowsButton = append(rowsButton, inlineKeyboardButton26)

			inlineKeyboardButton27 = append(inlineKeyboardButton27, tgbotapi.NewInlineKeyboardButtonData(cons.ContestFire.String(), string(cons.ContestFire)))
			rowsButton = append(rowsButton, inlineKeyboardButton27)

			inlineKeyboardButton28 = append(inlineKeyboardButton28, tgbotapi.NewInlineKeyboardButtonData(cons.ContestTrafficLight.String(), string(cons.ContestTrafficLight)))
			rowsButton = append(rowsButton, inlineKeyboardButton28)

			inlineKeyboardButton29 = append(inlineKeyboardButton29, tgbotapi.NewInlineKeyboardButtonData(cons.ContestSummerPalette.String(), string(cons.ContestSummerPalette)))
			rowsButton = append(rowsButton, inlineKeyboardButton29)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.CompleteApplication: %w", err)
			}
		}

	case botcommand.SelectProject:

		msg := tgbotapi.NewMessage(id, message, styleText)

		msg.ReplyMarkup = keyboardApplicationStart

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SelectProject: %w", err)
		}

		body := make([]string, 3)
		body = append(body, "В любой момент вы можете отменить заявку, нажав \"Отмена\"")
		body = append(body, "")
		body = append(body, fmt.Sprintf("Для продолжения заполнения заявки, необходимо дать согласие на обработку персональных данных.\n Ознакомиться с пользователським соглашением и политикой конфидециальности\n можно по ссылке %s", os.Getenv("PRIVACY_POLICY_TERMS_CONDITIONS")))
		text := strings.Join(body, "\n")

		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.Agree), cons.Agree.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg = tgbotapi.NewMessage(id, text, cons.StyleTextCommon)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SelectProject: %w", err)
		}

	case botcommand.SelectDegree:

		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton3 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.Degree1.String(), string(cons.Degree1)))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.Degree2.String(), string(cons.Degree2)))
		rowsButton = append(rowsButton, inlineKeyboardButton2)

		inlineKeyboardButton3 = append(inlineKeyboardButton3, tgbotapi.NewInlineKeyboardButtonData(cons.Degree3.String(), string(cons.Degree3)))
		rowsButton = append(rowsButton, inlineKeyboardButton3)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SelectDegree: %w", err)
		}

	case botcommand.GetDiploma:

		ctx := context.Background()
		conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("Connect to db is fail: %v", err.Error()))
		}
		defer conn.Close(ctx)

		rows, er := conn.Query(ctx, "SELECT requisition_number, start_date FROM \"certificates\" WHERE user_id=$1 ORDER BY requisition_number", id)
		if er != nil {
			zrlog.Error().Msg(fmt.Sprintf("func GetRequisitionForUser(), query to db is failed: %v", er))
		}

		var number int64
		var date int64
		var rowsButton [][]tgbotapi.InlineKeyboardButton

		//Add blank the inlineRow1KeyboardButtons
		inlineRow1KeyboardButtons := make([]tgbotapi.InlineKeyboardButton, 0, 0)
		rowsButton = append(rowsButton, inlineRow1KeyboardButtons)

		for rows.Next() {

			err := rows.Scan(&number, &date)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("botcommand.GetDiploma: %v", rows.Err().Error()))
			}

			inlineRowKeyboardButtons := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineRowKeyboardButtons = append(inlineRowKeyboardButtons, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("№%v от %v", number, unixNanoToDateString(date)), strconv.Itoa(int(number))))
			rowsButton = append(rowsButton, inlineRowKeyboardButtons)
		}

		rows.Close()

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		if number != 0 {
			message = "Выберите заявку:"
		} else {
			message = "У Вас нет грамот (дипломов) для получения!\n\nВозможно, Вы их уже получили (на почту или в телегрмме)."
		}

		messageRM := tgbotapi.NewMessage(id, message, styleText)
		messageRM.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(messageRM); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GetDiploma: %w", err)
		}

	case botcommand.GetPublicationDate:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GetPublicationDate: %w", err)
		}

	case botcommand.GetPublicationLink:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueClosingApplication

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.GetPublicationLink: %w", err)
		}

	case botcommand.WaitingForAcceptance:

		msg := tgbotapi.NewMessage(id, message, cons.StyleTextCommon)

		if !userPolling.Get(id).Agree {
			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.Agree), cons.Agree.String()))
			rowsButton = append(rowsButton, inlineKeyboardButton1)
			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg.ReplyMarkup = inlineKeyboardMarkup
		} else {
			msg.ReplyMarkup = keyboardApplicationStart
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.WaitingForAcceptance: (%T %+v) %w", err, err, err)
		}

	case botcommand.ContinueDataPolling:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardContinueClosingApplication
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling1
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.ContinueDataPolling: %w", err)
		}

	case botcommand.RecordToDB:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.RecordToDB: %w", err)
		}

	case botcommand.AskAge:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardContinueDataPolling2

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.AskAge: %w", err)
		}

	case botcommand.SelectFNPLeader:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardContinueDataPolling2
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SelectFNPLeader: %w", err)
		}

	case botcommand.SelectDocumentType:

		var rowsButton [][]tgbotapi.InlineKeyboardButton

		inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
		inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

		inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(string(cons.Certificate), cons.Certificate.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton1)

		inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(string(cons.CertificateAndDiploma), cons.CertificateAndDiploma.String()))
		rowsButton = append(rowsButton, inlineKeyboardButton2)

		inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = inlineKeyboardMarkup

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.SelectDocumentType: %w", err)
		}

	case botcommand.SelectPlaceDeliveryOfDocuments:

		if !thisIsAdmin(id) {
			var rowsButton [][]tgbotapi.InlineKeyboardButton

			inlineKeyboardButton1 := make([]tgbotapi.InlineKeyboardButton, 0, 1)
			inlineKeyboardButton2 := make([]tgbotapi.InlineKeyboardButton, 0, 1)

			inlineKeyboardButton1 = append(inlineKeyboardButton1, tgbotapi.NewInlineKeyboardButtonData(cons.PlaceDeliveryOfDocuments1, cons.PlaceDeliveryOfDocuments1))
			rowsButton = append(rowsButton, inlineKeyboardButton1)

			inlineKeyboardButton2 = append(inlineKeyboardButton2, tgbotapi.NewInlineKeyboardButtonData(cons.PlaceDeliveryOfDocuments2, cons.PlaceDeliveryOfDocuments2))
			rowsButton = append(rowsButton, inlineKeyboardButton2)

			inlineKeyboardMarkup := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rowsButton}

			msg := tgbotapi.NewMessage(id, message, styleText)
			msg.ReplyMarkup = inlineKeyboardMarkup

			if _, err := bot.Send(msg); err != nil {
				return fmt.Errorf("sentToTelegram(), botcommand.SelectPlaceDeliveryOfDocuments: %w", err)
			}
		}

	case botcommand.CheckData:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if !thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirm
		} else {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CheckData: %w", err)
		}

		message = UserDataToStringForTelegramm(id)

		msg = tgbotapi.NewMessage(id, message, cons.StyleTextHTML)

		if !thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardConfirm
		} else {
			msg.ReplyMarkup = keyboardConfirmForAdmin
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CheckData: %w", err)
		}

	case botcommand.CheckPDFFiles:

		msg := tgbotapi.NewMessage(id, message, styleText)
		msg.ReplyMarkup = keyboardConfirmAndSendForAdmin

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.CheckPDFFiles: %w", err)
		}

	case botcommand.Undefined:

		msg := tgbotapi.NewMessage(id, message, styleText)

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegram(), botcommand.Undefined: %w", err)
		}

	case botcommand.Settings:
	}

	return nil
}

func sentToTelegramPDF(bot *tgbotapi.BotAPI, id int64, pdfPath, fileID string, command botcommand.BotCommand) error {
	var msg tgbotapi.DocumentConfig

	switch command {
	case botcommand.SelectProject:

		if fileID != "" {
			msg = tgbotapi.NewDocumentShare(id, fileID)
		} else {
			msg = tgbotapi.NewDocumentUpload(id, pdfPath)
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

		if fileID != "" {
			msg = tgbotapi.NewDocumentShare(id, fileID)
		} else {
			msg = tgbotapi.NewDocumentUpload(id, pdfPath)
		}

		if thisIsAdmin(id) {
			msg.ReplyMarkup = keyboardAdminMainMenue
		} else {
			msg.ReplyMarkup = keyboardMainMenue
		}

		if _, err := bot.Send(msg); err != nil {
			return fmt.Errorf("sentToTelegramPDF(), botcommand.SelectProject: %w", err)
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

func AddRequisition(ctx context.Context, userID int64, dbpool *pgxpool.Pool) error {
	userData := userPolling.Get(userID)

	row, err := dbpool.Query(ctx, fmt.Sprintf("insert into %s (user_id, contest, user_fnp, group_age, name_institution, locality, naming_unit, publication_title, leader_fnp, email, document_type, place_delivery_of_document, diploma, start_date, expiration, close_date) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) returning requisition_number", userData.TableDB), userID, userData.Contest, userData.FNP, userData.Age, userData.NameInstitution, userData.Locality, userData.NamingUnit, userData.PublicationTitle, userData.LeaderFNP, userData.Email, userData.DocumentType, userData.PlaceDeliveryDocuments, userData.Diploma, time.Now().UnixNano(), int64(time.Now().Add(172800*time.Second).UnixNano()), 0)
	if err != nil {
		return fmt.Errorf("func AddRequisition(), query to db is failed: %w", err)
	}

	if row.Next() {
		var requisitionNumber int

		err := row.Scan(&requisitionNumber)
		if err != nil {
			return fmt.Errorf("func AddRequisition(), scan datas of row is failed %w", err)
		}

		userPolling.Set(userID, enumapplic.RequisitionNumber, "", int64(requisitionNumber))

		if userData.Diploma {
			var diplomaNumber int

			row, err := dbpool.Query(ctx, "insert into diplomas (requisition_number) values ($1) returning diploma_number", requisitionNumber)
			if err != nil {
				return fmt.Errorf("func AddRequisition(), query to db is failed: %w", err)
			}

			if row.Next() {
				err := row.Scan(&diplomaNumber)
				if err != nil {
					return fmt.Errorf("func AddRequisition(), scan datas of row is failed %w", err)
				}

				userPolling.Set(userID, enumapplic.DiplomaNumber, "", int64(diplomaNumber))
			}
		}
	}

	return row.Err()
}

func GetRequisitionForAdmin(ctx context.Context, adminID int64, requisitionNumber int64, degree int, tableDB, publicationDate, publicationLink string, dbpool *pgxpool.Pool) error {

	var userID int
	var fnp string
	var age string
	var nameInstitution string
	var locality string
	var namingUnit string
	var publicationTitle string
	var leaderFNP string
	var email string
	var contest string
	var documentType string
	var diploma bool
	var diplomaNumber int64
	var placeDeliveryOfDocument string

	row, err := dbpool.Query(ctx, fmt.Sprintf("SELECT user_id, user_fnp, COALESCE(group_age, ''), name_institution, locality, naming_unit, publication_title, leader_fnp, email, contest, document_type, place_delivery_of_document, diploma, COALESCE(diploma_number, 0) FROM %s LEFT JOIN diplomas ON %s.requisition_number=diplomas.requisition_number WHERE %s.requisition_number = $1", tableDB, tableDB, tableDB), requisitionNumber)
	if err != nil {
		return fmt.Errorf("func GetRequisitionForAdmin(), query to db is failed: %w", err)
	}

	if row.Next() {
		err := row.Scan(&userID, &fnp, &age, &nameInstitution, &locality,
			&namingUnit, &publicationTitle, &leaderFNP, &email, &contest, &documentType,
			&placeDeliveryOfDocument, &diploma, &diplomaNumber)

		if err != nil {
			return fmt.Errorf("func GetRequisitionForAdmin(), scan datas of row is failed %w", err)
		}

		closingRequisition.Set(adminID, enumapplic.UserID, strconv.Itoa(userID))
		closingRequisition.Set(adminID, enumapplic.FNP, fnp)
		closingRequisition.Set(adminID, enumapplic.Age, age)
		closingRequisition.Set(adminID, enumapplic.NameInstitution, nameInstitution)
		closingRequisition.Set(adminID, enumapplic.Locality, locality)
		closingRequisition.Set(adminID, enumapplic.NamingUnit, namingUnit)
		closingRequisition.Set(adminID, enumapplic.PublicationTitle, publicationTitle)
		closingRequisition.Set(adminID, enumapplic.FNPLeader, leaderFNP)
		closingRequisition.Set(adminID, enumapplic.Email, email)
		closingRequisition.Set(adminID, enumapplic.Contest, contest)
		closingRequisition.Set(adminID, enumapplic.DocumentType, documentType)
		closingRequisition.Set(adminID, enumapplic.PlaceDeliveryOfDocuments, placeDeliveryOfDocument)
		closingRequisition.Set(adminID, enumapplic.RequisitionNumber, fmt.Sprintf("%v", requisitionNumber))
		closingRequisition.Set(adminID, enumapplic.PublicationLink, publicationLink)
		closingRequisition.Set(adminID, enumapplic.PublicationDate, publicationDate)
		closingRequisition.Set(adminID, enumapplic.Degree, strconv.Itoa(degree))
		closingRequisition.Set(adminID, enumapplic.TableDB, cons.Certificate.String())
		closingRequisition.Set(adminID, enumapplic.Diploma, strconv.FormatBool(diploma))

		if diploma {
			closingRequisition.Set(adminID, enumapplic.DiplomaNumber, strconv.Itoa(int(diplomaNumber)))
		}

	}

	return row.Err()
}

func GetRequisitionForUser(ctx context.Context, userID, requisitionNumber int64, dbpool *pgxpool.Pool) error {

	var startDate int64
	var fnp string
	var age string
	var nameInstitution string
	var locality string
	var namingUnit string
	var publicationTitle string
	var publicationLink string
	var publicationDate int64
	var degree int
	var leaderFNP string
	var email string
	var contest string
	var documentType string
	var diploma bool
	var diplomaNumber int64

	row, err := dbpool.Query(ctx, fmt.Sprintf("SELECT start_date, user_fnp, COALESCE(group_age, '') AS age, name_institution, locality, naming_unit, publication_title, COALESCE(reference, '') AS reference, publication_date, degree,leader_fnp, email, contest, document_type, diploma, COALESCE(diploma_number, 0) AS diploma_number FROM \"%s\" LEFT JOIN \"diplomas\" ON %s.requisition_number=diplomas.requisition_number WHERE %s.user_id=$1 AND %s.requisition_number=$2 ORDER BY %s.requisition_number", cons.Certificate.String(), cons.Certificate.String(), cons.Certificate.String(), cons.Certificate.String(), cons.Certificate.String()), userID, requisitionNumber)
	if err != nil {
		return fmt.Errorf("func GetRequisitionForUser(), query to db is failed: %W", err)
	}

	for row.Next() {

		err := row.Scan(&startDate, &fnp, &age, &nameInstitution, &locality, &namingUnit, &publicationTitle, &publicationLink, &publicationDate, &degree, &leaderFNP, &email, &contest, &documentType, &diploma, &diplomaNumber)

		if err != nil {
			return fmt.Errorf("func GetRequisitionForUser(), scan datas of row is failed %w", err)
		}

		// if userID == 0 && publicationDate != 0 {
		// 	sent = true
		// }

		// if userID == 0 || userID != userid {
		// 	return userID, sent, row.Err()
		// }

		dateString := unixNanoToDateString(publicationDate)

		userPolling.Set(userID, enumapplic.FNP, fnp, 0)
		userPolling.Set(userID, enumapplic.Age, age, 0)
		userPolling.Set(userID, enumapplic.NameInstitution, nameInstitution, 0)
		userPolling.Set(userID, enumapplic.Locality, locality, 0)
		userPolling.Set(userID, enumapplic.NamingUnit, namingUnit, 0)
		userPolling.Set(userID, enumapplic.PublicationTitle, publicationTitle, 0)
		userPolling.Set(userID, enumapplic.FNPLeader, leaderFNP, 0)
		userPolling.Set(userID, enumapplic.Email, email, 0)
		userPolling.Set(userID, enumapplic.Contest, contest, 0)
		userPolling.Set(userID, enumapplic.DocumentType, documentType, 0)
		userPolling.Set(userID, enumapplic.RequisitionNumber, "", requisitionNumber)
		userPolling.Set(userID, enumapplic.RequisitionStartDate, "", startDate)
		userPolling.Set(userID, enumapplic.PublicationLink, publicationLink, 0)
		userPolling.Set(userID, enumapplic.PublicationDate, dateString, 0)
		userPolling.Set(userID, enumapplic.Degree, strconv.Itoa(degree), 0)
		userPolling.Set(userID, enumapplic.TableDB, cons.Certificate.String(), 0)
		userPolling.Set(userID, enumapplic.Diploma, strconv.FormatBool(diploma), 0)

		if diploma {
			userPolling.Set(userID, enumapplic.DiplomaNumber, "", diplomaNumber)
		}
	}

	return row.Err()
}

func UpdateRequisition(ctx context.Context, admin, cleanOut bool, requisitionNumber int64, tableDB string, degree int, publicationLink, publicationDate string, dbpool *pgxpool.Pool) (err error) {
	var query string

	dateOfPublication := dateStringToUnixNano(publicationDate)

	switch admin {
	case true:

		if cleanOut {
			query = fmt.Sprintf("UPDATE %s SET reference='%v', publication_date='%v', close_date='%v', degree='%v', email='%v',user_fnp='%v',leader_fnp='%v',user_id='%v' WHERE requisition_number=$1 RETURNING user_id", tableDB, publicationLink, dateOfPublication, time.Now().UnixNano(), degree, "", "", "", 0)
		} else {
			query = fmt.Sprintf("UPDATE %s SET reference='%v', publication_date='%v', close_date='%v', degree='%v' WHERE requisition_number=$1 RETURNING user_id", tableDB, publicationLink, dateOfPublication, time.Now().UnixNano(), degree)
		}

	default:
		query = fmt.Sprintf("UPDATE %s SET email='',user_fnp='',leader_fnp='',user_id=0 WHERE requisition_number=$1 RETURNING user_id", tableDB)
	}

	row, err := dbpool.Query(ctx, query, requisitionNumber)
	if err != nil {
		return fmt.Errorf("query UPDATE to db is failed: %w", err)
	}

	row.Next()

	return row.Err()
}

func FillInCertificatesPDFForms(wg *sync.WaitGroup, userID int64) {
	defer wg.Done()

	var userData cache.DataPolling

	if thisIsAdmin(userID) {
		userData = closingRequisition.Get(userID).UserData
	} else {
		userData = userPolling.Get(userID)
	}

	var x float64
	var y float64 = 305.0
	var step float64 = 15.0
	var widthText float64
	var centerX float64 = 297.5
	var path string
	var degree string

	if strings.TrimSpace(userData.LeaderFNP) != "" {
		path = "./external/imgs/%s_%s_curator.jpg"
	} else {
		path = "./external/imgs/%s_%s.jpg"
	}

	boilerplatePDFPath := fmt.Sprintf(path, contests[userData.Contest], cons.Certificate.String())

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	err := pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), AddTTFFont: %v", err.Error()))
	}

	pdf.AddPage()

	rect := &gopdf.Rect{W: 595, H: 842} // Page size A4 format
	err = pdf.Image(boilerplatePDFPath, 0, 0, rect)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Image(): %v", err))
	}

	// 1. Degree

	pdf.SetXY(235, 211)
	pdf.SetTextColorCMYK(0, 100, 100, 0) // Red
	err = pdf.SetFont("TelegraphLine", "", 24)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), SetFont: %v", err.Error()))
	}

	switch userData.Degree {
	case 1:
		degree = "I"
	case 2:
		degree = "II"
	case 3:
		degree = "III"
	}

	err = pdf.Text(degree)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(), degree: %v", err.Error()))
	}

	// 2. Requisition number
	x = 101

	pdf.SetXY(x, 251)
	pdf.SetTextColorCMYK(58, 46, 41, 94) // black
	err = pdf.SetFont("TelegraphLine", "", 14)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont(): %v", err.Error()))
	}

	err = pdf.Text(fmt.Sprintf("%v", userData.RequisitionNumber))

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(): %v", err.Error()))
	}

	// 3. Name

	pdf.SetTextColorCMYK(0, 100, 100, 0) // Red
	err = pdf.SetFont("TelegraphLine", "", 18)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont(): %v", err.Error()))
	}

	widthText, err = pdf.MeasureTextWidth(userData.FNP)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(): %v", err.Error()))
	}

	x = centerX - widthText/2

	if widthText > maxWidthPDF {
		var arrayText []string

		arrayText, err = pdf.SplitText(userData.FNP, maxWidthPDF)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(): %v", err.Error()))
		}

		y = pdf.GetY() + 2*step

		for _, t := range arrayText {
			widthText, err = pdf.MeasureTextWidth(t)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
			}

			x = centerX - widthText/2

			pdf.SetXY(x, y)
			err = pdf.Text(t)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(t): %v", err.Error()))
			}
			y = y + step
		}
	} else {
		pdf.SetXY(x, 275)
		err = pdf.Text(userData.FNP)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.FNP): %v", err.Error()))
		}
	}

	// 4. Age
	if strings.TrimSpace(userData.Age) != "0" {

		var ageString string
		var ending string
		var symbols string

		groupAge := strings.TrimSpace(userData.Age)
		ageString = groupAge

		contain1 := strings.Contains(groupAge, "лет")
		contain2 := strings.Contains(groupAge, "года")
		contain3 := strings.Contains(groupAge, "год")
		contain4 := strings.Contains(groupAge, "годиков")
		contain5 := strings.Contains(groupAge, "годика")
		contain6 := strings.Contains(groupAge, "г")
		contain7 := strings.Contains(groupAge, "г.")

		if !contain1 && !contain2 && !contain3 && !contain4 && !contain5 && !contain6 && !contain7 {
			var age int
			if len(groupAge) > 2 {
				for i := 1; i < len(groupAge)-1; i++ {
					symbols = string(groupAge[len(groupAge)-i:])
					age, err = strconv.Atoi(symbols)

					if err != nil {
						if i > 1 {
							symbols = string(groupAge[len(groupAge)-i+1:])
							age, _ = strconv.Atoi(symbols)
						} else {
							age = 0
						}
						break
					}
				}

			} else {
				age, err = strconv.Atoi(groupAge)

				if err != nil {
					age = 0
				}
			}

			ending = convertAgeToString(age)
			ageString = fmt.Sprintf("%v %v", ageString, ending)
		}

		widthText, err = pdf.MeasureTextWidth(ageString)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(age_string): %v", err))
		}

		x = centerX - widthText/2
		y = pdf.GetY() + 1.5*step

		pdf.SetXY(x, y)
		err = pdf.Text(ageString)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(age_string): %v", err))
		}
	}

	// 5. Name institution

	y = pdf.GetY() + 2*step

	pdf.SetTextColorCMYK(58, 46, 41, 94) // black
	err = pdf.SetFont("TelegraphLine", "", 16)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont() Name institution: %v", err))
	}

	widthText, err = pdf.MeasureTextWidth(userData.NameInstitution)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(userData.NameInstitution): %v", err))
	}

	if widthText > maxWidthPDF-80 {
		arrayText := strings.Split(userData.NameInstitution, " ")

		var t string

		for _, word := range arrayText {
			t = fmt.Sprintf("%s %s", t, word)

			widthText, err = pdf.MeasureTextWidth(t)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(t) name institution: %v", err.Error()))
			}

			if widthText > maxWidthPDF-80 {
				widthText, err = pdf.MeasureTextWidth(userData.NameInstitution[:len(t)-1])

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(textPart1) name institution: %v", err.Error()))
				}

				x = centerX - widthText/2

				textPart1 := userData.NameInstitution[:len(t)-1]

				pdf.SetXY(x, y)
				err = pdf.Text(textPart1)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(textPart1): %v", err.Error()))
				}
				y = y + step

				textPart2 := userData.NameInstitution[len(t)-1:]

				widthText, err = pdf.MeasureTextWidth(textPart2)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(textPart2) name institution: %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				err = pdf.Text(textPart2)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(textPart2): %v", err.Error()))
				}
				y += step

				zrlog.Info().Msg("Split long name institution")

				break
			}
		}
	} else {
		y = pdf.GetY() + 2*step
		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(userData.NameInstitution)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.NameInstitution): %v", err.Error()))
		}
	}

	// 6. Locality

	y = pdf.GetY() + 1.5*step

	pdf.SetTextColorCMYK(58, 46, 41, 94) // black

	widthText, err = pdf.MeasureTextWidth(userData.Locality)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(userData.Locality): %v", err.Error()))
	}

	if widthText > maxWidthPDF-80 {
		arrayText := strings.Split(userData.Locality, " ")

		var t string

		for _, word := range arrayText {
			t = fmt.Sprintf("%s %s", t, word)

			widthText, err = pdf.MeasureTextWidth(t)

			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(t) locality: %v", err.Error()))
			}

			if widthText > maxWidthPDF-80 {
				textPart1 := userData.Locality[:len(t)-1]

				widthText, err = pdf.MeasureTextWidth(textPart1)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(textPart1) locality: %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				err = pdf.Text(textPart1)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(textPart1): %v", err.Error()))
				}
				y = y + step

				textPart2 := userData.Locality[len(t)-1:]

				widthText, err = pdf.MeasureTextWidth(textPart2)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(textPart2) locality: %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				err = pdf.Text(textPart2)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(textPart2) locality: %v", err.Error()))
				}
				y += step
				break
			}
		}
	} else {
		x = centerX - widthText/2

		pdf.SetXY(x, y)
		err = pdf.Text(userData.Locality)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.Locality): %v", err.Error()))
		}
	}

	// 7. Naming unit

	pdf.SetXY(152, 622)
	pdf.SetTextColorCMYK(58, 46, 41, 94) // black

	err = pdf.SetFont("TelegraphLine", "", 18)
	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SetFont() Name institution: %v", err))
	}

	err = pdf.Text(userData.NamingUnit)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.NamingUnit): %v", err.Error()))
	}

	// 8. Publication title

	pdf.SetXY(194, 646)
	pdf.SetTextColorCMYK(58, 46, 41, 94) // black

	err = pdf.Text(userData.PublicationTitle)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.PublicationTitle): %v", err.Error()))
	}

	// 9. Leader's FNP

	if strings.TrimSpace(userData.LeaderFNP) != "" {
		y = 668
		x = 194

		pdf.SetXY(x, y)

		var arrayLeaders []string
		var maxWidth float64

		contain := strings.Contains(userData.LeaderFNP, cons.Comma)

		switch contain {
		case true:

			arrayLeaders = strings.Split(userData.LeaderFNP, cons.Comma)

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
						zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(leader, maxWidth) Leader's FNP: %v", err.Error()))
					}

					for k, t := range arrayText {
						widthText, err = pdf.MeasureTextWidth(t)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(t) Leader's FNP: %v", err.Error()))
						}

						if i > 0 || k > 0 { // Second leader or second part part first leader
							x = 55
						}

						pdf.SetXY(x, y)
						err = pdf.Text(t)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(t) Leader's FNP: %v", err.Error()))
						}

						y = y + 1.2*step
					}
				} else {
					if i > 0 { // Second leader
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
			widthText, err = pdf.MeasureTextWidth(userData.LeaderFNP)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.MeasureTextWidth(userData.LeaderFNP): %v", err.Error()))
			}

			if widthText > maxWidth {
				var arrayText []string

				arrayText, err = pdf.SplitText(userData.LeaderFNP, maxWidth)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.SplitText(userData.LeaderFNP, maxWidth): %v", err.Error()))
				}

				for k, t := range arrayText {
					if k > 0 {
						x = 55
					}

					pdf.SetXY(x, y)
					err = pdf.Text(t)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(t): %v", err.Error()))
					}

					y = y + 1.2*step
				}
			} else {
				pdf.SetXY(x, y)
				err = pdf.Text(userData.LeaderFNP)
				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.LeaderFNP): %v", err.Error()))
				}
			}
		}
	}

	// 10. Publication date

	pdf.SetXY(426, 718)
	pdf.SetTextColorCMYK(58, 46, 41, 94) // black

	err = pdf.Text(userData.PublicationDate)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.PublicationDate): %v", err.Error()))
	}

	// 11. Publication link

	pdf.SetXY(50, 740)
	pdf.SetTextColorCMYK(58, 46, 41, 94) // black

	err = pdf.Text(userData.PublicationLink)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.Text(userData.PublicationLink): %v", err.Error()))
	}

	path = fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.Certificate), userData.RequisitionNumber)

	err = pdf.WritePdf(path)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(), pdf.WritePdf(path): %v", err.Error()))
	}

	if thisIsAdmin(userID) {
		closingRequisition.Set(userID, enumapplic.File, path)
	} else {
		userPolling.Set(userID, enumapplic.File, path, 0)
	}

	err = pdf.Close()

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("func FillInCertificatesPDFForms(),  pdf.Close(): %v", err.Error()))
	}
}

func FillInDiplomasPDFForms(wg *sync.WaitGroup, userID int64) {
	defer wg.Done()

	var x float64
	var y float64 = 305
	var step float64 = 15
	var widthText float64
	var centerX float64 = 297.5
	var degree string
	var userData cache.DataPolling

	if thisIsAdmin(userID) {
		userData = closingRequisition.Get(userID).UserData
	} else {
		userData = userPolling.Get(userID)
	}

	if userData.Diploma {
		boilerplatePDFPath := fmt.Sprintf("./external/imgs/%s_%s.jpg", contests[userData.Contest], cons.Diploma.String())

		pdf := gopdf.GoPdf{}
		pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

		err := pdf.AddTTFFont("TelegraphLine", "./external/fonts/ttf/TelegraphLine.ttf")
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.AddTTFFont(): %v", err.Error()))
		}

		pdf.AddPage()
		rect := &gopdf.Rect{W: 595, H: 842} // Page size A4 format
		err = pdf.Image(boilerplatePDFPath, 0, 0, rect)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Image(boilerplatePDFPath, 0, 0, rect): %v", err.Error()))
		}

		// 1. Diploma number

		pdf.SetTextColorCMYK(58, 46, 41, 94) // black
		err = pdf.SetFont("TelegraphLine", "", 14)

		// 2. Requisition number
		x = 90

		pdf.SetXY(x, 242)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		err = pdf.Text(fmt.Sprintf("%v", userData.DiplomaNumber))

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.DiplomaNumber): %v", err.Error()))
		}

		// 2. Leader's FNP
		pdf.SetTextColorCMYK(0, 100, 100, 0) // Red
		err = pdf.SetFont("TelegraphLine", "", 18)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		var arrayLeaders []string

		y = 262

		contain := strings.Contains(userData.LeaderFNP, cons.Comma)

		switch contain {
		case true:

			arrayLeaders = strings.Split(userData.LeaderFNP, cons.Comma)

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
						err = pdf.Text(t)
						if err != nil {
							zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(t): %v", err.Error()))
						}
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

				arrayText, err = pdf.SplitText(userData.LeaderFNP, maxWidthPDF)
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
					err = pdf.Text(t)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(t): %v", err.Error()))
					}
					y = y + 1.2*step
				}
			} else {
				widthText, err = pdf.MeasureTextWidth(userData.LeaderFNP)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.LeaderFNP): %v", err.Error()))
				}

				x = centerX - widthText/2

				pdf.SetXY(x, y)
				err = pdf.Text(userData.LeaderFNP)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.LeaderFNP): %v", err.Error()))
				}
			}
		}

		// 3. Name institution

		pdf.SetTextColorCMYK(58, 46, 41, 94) // black
		err = pdf.SetFont("TelegraphLine", "", 16)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), Name institution, pdf.SetFont(): %v", err.Error()))
		}

		y = pdf.GetY() + 1.5*step

		widthText, err = pdf.MeasureTextWidth(userData.NameInstitution)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.NameInstitution): %v", err.Error()))
		}

		if widthText > maxWidthPDF-80 {
			arrayText := strings.Split(userData.NameInstitution, " ")

			var t string

			for _, word := range arrayText {
				t = fmt.Sprintf("%s %s", t, word)

				widthText, err = pdf.MeasureTextWidth(t)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t) Name institution: %v", err.Error()))
				}

				if widthText > maxWidthPDF-80 {
					textPart1 := userData.NameInstitution[:len(t)-1]

					widthText, err = pdf.MeasureTextWidth(textPart1)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(textPart1) Name institution: %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					err = pdf.Text(textPart1)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(textPart1) Name institution: %v", err.Error()))
					}
					y = y + step

					textPart2 := userData.NameInstitution[len(t)-1:]

					widthText, err = pdf.MeasureTextWidth(textPart2)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(textPart2) Name institution: %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					err = pdf.Text(textPart2)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(textPart2) Name institution: %v", err.Error()))
					}
					y = y + step
					break
				}
			}
		} else {
			y = pdf.GetY() + 2*step
			x = centerX - widthText/2

			pdf.SetXY(x, y)
			err = pdf.Text(userData.NameInstitution)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.NameInstitution): %v", err.Error()))
			}
		}

		// 4. Locality
		pdf.SetTextColorCMYK(58, 46, 41, 94) // black
		err = pdf.SetFont("TelegraphLine", "", 16)
		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont() Locality: %v", err.Error()))
		}

		y = pdf.GetY() + 1.5*step

		widthText, err = pdf.MeasureTextWidth(userData.Locality)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(usersRequisition.Locality): %v", err.Error()))
		}

		if widthText > maxWidthPDF-80 {
			arrayText := strings.Split(userData.Locality, " ")

			var t string

			for _, word := range arrayText {
				t = fmt.Sprintf("%s %s", t, word)

				widthText, err = pdf.MeasureTextWidth(t)

				if err != nil {
					zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(t): %v", err.Error()))
				}

				if widthText > maxWidthPDF-80 {
					textPart1 := userData.Locality[:len(t)-1]

					widthText, err = pdf.MeasureTextWidth(textPart1)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(textPart1) locality: %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					err = pdf.Text(textPart1)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(textPart1) locality: %v", err.Error()))
					}
					y = y + step

					textPart2 := userData.Locality[len(t)-1:]

					widthText, err = pdf.MeasureTextWidth(textPart2)

					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.MeasureTextWidth(textPart2) locality: %v", err.Error()))
					}

					x = centerX - widthText/2

					pdf.SetXY(x, y)
					err = pdf.Text(textPart2)
					if err != nil {
						zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(textPart2) locality: %v", err.Error()))
					}
					y = y + step
					break
				}
			}
		} else {
			x = centerX - widthText/2

			pdf.SetXY(x, y)
			err = pdf.Text(userData.Locality)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.Locality): %v", err.Error()))
			}
		}

		// 5. FNP
		pdf.SetXY(142, 627)
		err = pdf.SetFont("TelegraphLine", "", 18)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.SetFont(): %v", err.Error()))
		}

		err = pdf.Text(userData.FNP)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.FNP): %v", err.Error()))
		}

		// 6. Naming unit

		pdf.SetXY(153, 653)

		err = pdf.Text(userData.NamingUnit)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.NamingUnit): %v", err.Error()))
		}

		// 7. Publication title

		pdf.SetXY(195, 674)

		err = pdf.Text(userData.PublicationTitle)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationTitle): %v", err.Error()))
		}

		// 8. Requisition number

		pdf.SetXY(139, 697)

		err = pdf.Text(fmt.Sprintf("%v", userData.RequisitionNumber))

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.RequisitionNumber): %v", err.Error()))
		}

		// 9. Degree

		var textDegree string

		switch userData.Degree {
		case 1:
			degree = "I"
			textDegree = fmt.Sprintf(",   %s", degree)
		case 2:
			degree = "II"
			x = 230.0
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

		// 10. Publication date

		pdf.SetXY(447, 736)

		err = pdf.Text(userData.PublicationDate)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationDate): %v", err.Error()))
		}

		// 11. Publication link

		pdf.SetXY(56, 757)

		err = pdf.Text(userData.PublicationLink)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.Text(usersRequisition.PublicationLink): %v", err.Error()))
		}

		path := fmt.Sprintf("./external/files/usersfiles/%s №%v.pdf", string(cons.Diploma), userData.RequisitionNumber)

		err = pdf.WritePdf(path)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func FillInDiplomasPDFForms(), pdf.WritePdf(path): %v", err.Error()))
		}

		if thisIsAdmin(userID) {
			closingRequisition.Set(userID, enumapplic.File, path)
		} else {
			userPolling.Set(userID, enumapplic.File, path, 0)
		}

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

	rect := &gopdf.Rect{W: 595, H: 842} // Page size A4 format
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

	err = pdf.CellWithOption(nil, fmt.Sprintf("Заявка №%v от %v ", usersRequisition.RequisitionNumber, formattedTime), cellOptionCaption)
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
	if strings.TrimSpace(usersRequisition.LeaderFNP) != "" {
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
	text := fmt.Sprintf("%s: \"%s\"", enumapplic.NamingUnit, usersRequisition.NamingUnit)
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
			err = pdf.Text(t)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(t): %v", err.Error()))
			}
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
	text = fmt.Sprintf("%s: \"%s\"", enumapplic.PublicationTitle, usersRequisition.PublicationTitle)
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
			err = pdf.Text(t)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(t): %v", err.Error()))
			}
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
	text = fmt.Sprintf("%s: %s", enumapplic.DocumentType, usersRequisition.DocumentType)
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
			err = pdf.Text(t)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(t): %v", err.Error()))
			}
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
	text = fmt.Sprintf("%s: %s", enumapplic.PlaceDeliveryOfDocuments, usersRequisition.PlaceDeliveryDocuments)
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
			err = pdf.Text(t)
			if err != nil {
				zrlog.Error().Msg(fmt.Sprintf("func ConvertRequisitionToPDF(), pdf.Text(t): %v", err.Error()))
			}
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

func SentEmail(to string, userID int64, toAdmin bool, requisition bool, subject string, files []string, message string) (bool, error) {

	userData := userPolling.Get(userID)

	if toAdmin && requisition {
		message = FormatUserDataToText(userData)
	}

	m := gomail.NewMessage()

	m.SetHeader("From", os.Getenv("BOT_EMAIL"))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)

	if toAdmin && requisition {
		m.Embed(userData.Photo)
	}

	if len(files) > 0 {
		temp := ""
		for _, path := range files {
			// When some files are the same
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
	d := gomail.NewDialer(os.Getenv("SMTP_SERVER"), 465, os.Getenv("BOT_EMAIL"), os.Getenv("BOT_EMAIL_PASSWORD"))

	if err := d.DialAndSend(m); err != nil {
		return false, err
	}

	return true, nil
}

func FormatUserDataToText(userData cache.DataPolling) string {

	var text string

	body := make([]string, 12)

	body = append(body, "<!DOCTYPE html><html lang=\"ru\"><body><dl>")

	body = append(body, "<style type=\"text/css\">BODY {margin: 0; /* Убираем отступы в браузере */}#toplayer {background: #F5FFFA; /* Цвет фона */height: 800px /* Высота слоя */}</style>")

	body = append(body, fmt.Sprintf("<div id=\"toplayer\"><dt><p><b>(%v). %s:</b></p></dt>", enumapplic.Contest.EnumIndex(), enumapplic.Contest.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.Contest))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", userData.FNP))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.Age.EnumIndex(), enumapplic.Age.String()))
	text = strings.Join(body, "\n")

	var ageString string

	if strings.TrimSpace(userData.Age) != cons.Zero {
		var ending string
		var symbols string

		groupAge := strings.TrimSpace(userData.Age)
		ageString = groupAge

		contain1 := strings.Contains(groupAge, "лет")
		contain2 := strings.Contains(groupAge, "года")
		contain3 := strings.Contains(groupAge, "год")
		contain4 := strings.Contains(groupAge, "годиков")
		contain5 := strings.Contains(groupAge, "годика")
		contain6 := strings.Contains(groupAge, "г")
		contain7 := strings.Contains(groupAge, "г.")

		if !contain1 && !contain2 && !contain3 && !contain4 && !contain5 && !contain6 && !contain7 {
			var age int
			var err error

			if len(groupAge) > 2 {
				for i := 1; i < len(groupAge)-1; i++ {
					symbols = string(groupAge[len(groupAge)-i:])
					age, err = strconv.Atoi(symbols)

					if err != nil {
						if i > 1 {
							symbols = string(groupAge[len(groupAge)-i+1:])
							age, _ = strconv.Atoi(symbols)
						} else {
							age = 0
						}
						break
					}
				}

			} else {
				age, err = strconv.Atoi(groupAge)

				if err != nil {
					age = 0
				}
			}

			ending = convertAgeToString(age)
			ageString = fmt.Sprintf("%v %v", ageString, ending)
		}

	} else {
		ageString = cons.NoAge
	}
	body = append(body, fmt.Sprintf("<dd><p>      %v</p></dd>", ageString))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NameInstitution.EnumIndex(), enumapplic.NameInstitution.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.NameInstitution))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.Locality.EnumIndex(), enumapplic.Locality.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p><dd>", userData.Locality))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.NamingUnit.EnumIndex(), enumapplic.NamingUnit.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.NamingUnit))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.PublicationTitle.EnumIndex(), enumapplic.PublicationTitle.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.PublicationTitle))
	text = strings.Join(body, "\n")

	if userData.LeaderFNP == "" {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v).</b> <s><i><b>%s:</b></i></s></p></dt>", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", "-"))
	} else {
		body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()))
		body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.LeaderFNP))
	}
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.Email.EnumIndex(), enumapplic.Email.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.Email))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b></p></dt>", enumapplic.DocumentType.EnumIndex(), enumapplic.DocumentType.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.DocumentType))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.PlaceDeliveryOfDocuments.EnumIndex(), enumapplic.PlaceDeliveryOfDocuments.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", userData.PlaceDeliveryDocuments))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.Photo.EnumIndex(), enumapplic.Photo.String()))
	body = append(body, fmt.Sprintf("<dd><p>      %s</p></dd>", "Прикреплена"))
	text = strings.Join(body, "\n")

	body = append(body, fmt.Sprintf("<dt><p><b>(%v). %s:</b><p></dt>", enumapplic.File.EnumIndex(), enumapplic.File.String()))
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
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.Contest.EnumIndex(), enumapplic.Contest.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Contest))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.FNP))
		text = strings.Join(body, "\n")

		var ageString string

		if strings.TrimSpace(usdata.Age) != "0" {
			var ending string
			var symbols string

			groupAge := strings.TrimSpace(usdata.Age)
			ageString = groupAge

			contain1 := strings.Contains(groupAge, "лет")
			contain2 := strings.Contains(groupAge, "года")
			contain3 := strings.Contains(groupAge, "год")
			contain4 := strings.Contains(groupAge, "годиков")
			contain5 := strings.Contains(groupAge, "годика")
			contain6 := strings.Contains(groupAge, "г")
			contain7 := strings.Contains(groupAge, "г.")

			if !contain1 && !contain2 && !contain3 && !contain4 && !contain5 && !contain6 && !contain7 {
				var age int
				var err error

				if len(groupAge) > 2 {
					for i := 1; i < len(groupAge)-1; i++ {
						symbols = string(groupAge[len(groupAge)-i:])
						age, err = strconv.Atoi(symbols)

						if err != nil {
							if i > 1 {
								symbols = string(groupAge[len(groupAge)-i+1:])
								age, _ = strconv.Atoi(symbols)
							} else {
								age = 0
							}
							break
						}
					}

				} else {
					age, err = strconv.Atoi(groupAge)

					if err != nil {
						age = 0
					}
				}

				ending = convertAgeToString(age)
				ageString = fmt.Sprintf("%v %v", ageString, ending)
			}

		} else {
			ageString = cons.NoAge
		}

		body = append(body, fmt.Sprintf("%v", "_________________________________"))

		if ageString == cons.NoAge {
			body = append(body, fmt.Sprintf("(%v). <s><i><b>%s:</b></i></s>", enumapplic.Age.EnumIndex(), enumapplic.Age.String()))
		} else {
			body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.Age.EnumIndex(), enumapplic.Age.String()))
		}

		body = append(body, fmt.Sprintf("      %v", ageString))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NameInstitution.EnumIndex(), enumapplic.NameInstitution.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.NameInstitution))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.Locality.EnumIndex(), enumapplic.Locality.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Locality))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.NamingUnit.EnumIndex(), enumapplic.NamingUnit.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.NamingUnit))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PublicationTitle.EnumIndex(), enumapplic.PublicationTitle.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.PublicationTitle))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		if usdata.LeaderFNP == "" {
			body = append(body, fmt.Sprintf("(%v). <s><i><b>%s:</b></i></s>", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()))
			body = append(body, fmt.Sprintf("      %s", "-"))
		} else {
			body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()))
			body = append(body, fmt.Sprintf("      %s", usdata.LeaderFNP))
		}
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.Email.EnumIndex(), enumapplic.Email.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.Email))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.DocumentType.EnumIndex(), enumapplic.DocumentType.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.DocumentType))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.PlaceDeliveryOfDocuments.EnumIndex(), enumapplic.PlaceDeliveryOfDocuments.String()))
		body = append(body, fmt.Sprintf("      %s", usdata.PlaceDeliveryDocuments))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.Photo.EnumIndex(), enumapplic.Photo.String()))
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")

		body = append(body, fmt.Sprintf("%v", "_________________________________"))
		body = append(body, fmt.Sprintf("(%v). <i><b>%s:</b></i>", enumapplic.File.EnumIndex(), enumapplic.File.String()))
		body = append(body, fmt.Sprintf("      %s", "Прикреплена"))
		text = strings.Join(body, "\n")
	} else {
		data := closingRequisition.Get(userID)
		body := make([]string, 12)

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.RequisitionNumber.String()))
		body = append(body, fmt.Sprintf("   %v", data.UserData.RequisitionNumber))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.Degree.String()))
		body = append(body, fmt.Sprintf("   %v", data.UserData.Degree))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PublicationDate.String()))
		body = append(body, fmt.Sprintf("   %s", data.UserData.PublicationDate))
		text = strings.Join(body, "\n")

		body = append(body, "_________________________________")
		body = append(body, fmt.Sprintf(" <i><b>%s:</b></i>", enumapplic.PublicationLink.String()))
		body = append(body, fmt.Sprintf("   %s", data.UserData.PublicationLink))
		text = strings.Join(body, "\n")
	}

	return text
}

func GetConciseDescription(contest string) string {
	var text string
	body := make([]string, 14)
	contests := make(map[string]struct{}, 0)

	contests[string(cons.ContestTitmouse)] = struct{}{}
	contests[string(cons.ContestMather)] = struct{}{}
	contests[string(cons.ContestFather)] = struct{}{}
	contests[string(cons.ContestAutumn)] = struct{}{}
	contests[string(cons.ContestWinter)] = struct{}{}
	contests[string(cons.ContestSnowflakes)] = struct{}{}
	contests[string(cons.ContestSnowman)] = struct{}{}
	contests[string(cons.ContestSymbol)] = struct{}{}
	contests[string(cons.ContestHearts)] = struct{}{}
	contests[string(cons.ContestSecrets)] = struct{}{}
	contests[string(cons.ContestBirdsFeeding)] = struct{}{}
	contests[string(cons.ContestShrovetide)] = struct{}{}
	contests[string(cons.ContestFable)] = struct{}{}
	contests[string(cons.ContestDefendersFatherland)] = struct{}{}
	contests[string(cons.ContestSpring)] = struct{}{}
	contests[string(cons.ContestMarchEighth)] = struct{}{}
	contests[string(cons.ContestEarth)] = struct{}{}
	contests[string(cons.ContestSpaceAdventures)] = struct{}{}
	contests[string(cons.ContestFeatheredFriends)] = struct{}{}
	contests[string(cons.ContestTheatricalBackstage)] = struct{}{}
	contests[string(cons.ContestOurFriends)] = struct{}{}
	contests[string(cons.ContestVictoryDay)] = struct{}{}
	contests[string(cons.ContestPrimroses)] = struct{}{}
	contests[string(cons.ContestMyFamily)] = struct{}{}
	contests[string(cons.ContestChildProtectionDay)] = struct{}{}
	contests[string(cons.ContestMotherRussia)] = struct{}{}
	contests[string(cons.ContestFire)] = struct{}{}
	contests[string(cons.ContestTrafficLight)] = struct{}{}
	contests[string(cons.ContestSummerPalette)] = struct{}{}

	_, ok := contests[contest]

	if ok {
		body = append(body, "<b>В заявке потребуется указать следующие данные:\n</b>")
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.Contest.EnumIndex(), enumapplic.Contest.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FNP.EnumIndex(), enumapplic.FNP.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.Age.EnumIndex(), enumapplic.Age.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NameInstitution.EnumIndex(), enumapplic.NameInstitution.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.Locality.EnumIndex(), enumapplic.Locality.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.NamingUnit.EnumIndex(), enumapplic.NamingUnit.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PublicationTitle.EnumIndex(), enumapplic.PublicationTitle.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.FNPLeader.EnumIndex(), enumapplic.FNPLeader.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.Email.EnumIndex(), enumapplic.Email.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.DocumentType.EnumIndex(), enumapplic.DocumentType.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.PlaceDeliveryOfDocuments.EnumIndex(), enumapplic.PlaceDeliveryOfDocuments.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.Photo.EnumIndex(), enumapplic.Photo.String()))
		body = append(body, fmt.Sprintf("(%v). <b>%s</b>", enumapplic.File.EnumIndex(), enumapplic.File.String()))
		body = append(body, "\n")
		body = append(body, fmt.Sprintf("Подробнее с условиями конкурса можно ознакомиться на сайте %s\n", os.Getenv("WEBSITE")))
		body = append(body, "\n")

		text = strings.Join(body, "\n")
	}

	return text
}

func downloadFile(filepath, url string) (err error) {
	// Create the file

	out, err := os.Create(filepath)
	// out, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0660)
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

func getFile(bot *tgbotapi.BotAPI, userID int64, fileID string, userData cache.DataPollingCache, botstateindex int64) {
	url, err := bot.GetFileDirectURL(fileID)

	if err != nil {
		zrlog.Error().Msg(fmt.Sprintf("bot can't get url of this file: %+v\n", err.Error()))
	} else {
		filename := path.Base(url)

		filePath := fmt.Sprintf("%s/%v_%v_%v", cons.FilePath, userID, botstateindex, filename)

		if botstateindex == botstate.AskPhoto.EnumIndex() {
			userPolling.Set(userID, enumapplic.Photo, filePath, 0)
			zrlog.Info().Msg(fmt.Sprintf("func getFile(), photo: %v\n", filePath))
		}

		if botstateindex == botstate.AskFile.EnumIndex() {
			userPolling.Set(userID, enumapplic.File, filePath, 0)
			zrlog.Info().Msg(fmt.Sprintf("func getFile(), file: %v\n", filePath))
		}

		err = downloadFile(filePath, url)

		if err != nil {
			zrlog.Error().Msg(fmt.Sprintf("func getFile(), bot can't download this file: %+v\n", err.Error()))
		}
	}
}

func deleteUserPolling(userID int64, userData cache.DataPollingCache) {
	userDP := userData.Get(userID)

	// delete user's files and datas in hashmap

	// removing file from the directory

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
		// When some files are the same
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

func unixNanoToDateString(date int64) string {
	t := time.Unix(0, date)

	dateString := t.Format(cons.TimeshortForm) //yyyy-mm-dd

	var d string
	var m string
	var y string

	sliceDate := strings.Split(dateString, "-")

	for k, v := range sliceDate {
		switch k {
		case 0:
			y = v
		case 1:
			m = v
		case 2:
			d = v
		}
	}

	return fmt.Sprintf("%s.%s.%s", d, m, y)
}

// func checkUsersIDCache(userID int64, bot *tgbotapi.BotAPI) {
// 	if tempUsersIDCache.Check(userID) {
// 		tempUsersIDCache.Delete(userID)

// 		adminID, err := strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)

// 		if err != nil {
// 			zrlog.Error().Msg(fmt.Sprintf("func checkUsersIDCache(), trconv.ParseInt(): %v\n", err))
// 		}

// 		err = sentToTelegram(bot, adminID, fmt.Sprintf("Можно закрывать заявки для пользователя %v!", userID), nil, cons.StyleTextCommon, botcommand.Undefined, "", "", false)

// 		if err != nil {
// 			zrlog.Error().Msg(fmt.Sprintf("func checkUsersIDCache(), sentToTelegramm() for admin: %v\n", err))
// 		}
// 	}
// }

func convertAgeToString(age int) string {
	var ending string

	ageString := strconv.Itoa(age)

	var numPrev string
	var numLast string
	var numbers [2]int

	if age >= 10 {
		numPrev = ageString[len(ageString)-2 : len(ageString)-1]
	} else {
		numPrev = "0"
	}

	numLast = ageString[len(ageString)-1:]

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

func validEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}
