package cons

import "github.com/kyokomi/emoji/v2"

// type TablesDB string
type InlineButton string

const (
	TelegrammURL               string       = "https://api.telegramm.org/bot"
	CertPaht                   string       = "./certs/cert.pem"
	KeyPath                    string       = "./certs/key.pem"
	MaxLengMsg                 int          = 4000
	StyleTextMarkdown          string       = "MarkdownV2"
	StyleTextCommon            string       = ""
	StyleTextHTML              string       = "HTML"
	PDFPath                    string       = "./external/pdf/"
	PDF                        bool         = true
	FilePath                   string       = "./external/files/usersfiles"
	ContestTitmouse            InlineButton = "Titmouse"
	ContestMather              InlineButton = "Mather"
	ContestFather              InlineButton = "Father"
	ContestAutumn              InlineButton = "Autumn"
	ContestWinter              InlineButton = "Winter"
	ContestSnowflakes          InlineButton = "Snowflakes"
	ContestSnowman             InlineButton = "Snowman"
	ContestSymbol              InlineButton = "Symbol"
	ContestHearts              InlineButton = "Hearts"
	ContestSecrets             InlineButton = "Secrets"
	ContestBirdsFeeding        InlineButton = "BirdsFeeding"
	ContestShrovetide          InlineButton = "Shrovetide"
	ContestFable               InlineButton = "Fable"
	ContestDefendersFatherland InlineButton = "DefendersFatherland"
	ContestSpring              InlineButton = "Spring"
	ContestMarchEighth         InlineButton = "MarchEighth"
	ContestEarth               InlineButton = "Earth"
	ContestSpaceAdventures     InlineButton = "SpaceAdventures"
	ContestFeatheredFriends    InlineButton = "FeatheredFriends"
	ContestTheatricalBackstage InlineButton = "TheatricalBackstage"
	ContestOurFriends          InlineButton = "OurFriends"
	ContestPrimroses           InlineButton = "Primroses"
	ContestVictoryDay          InlineButton = "VictoryDay"
	ContestMyFamily            InlineButton = "MyFamily"
	ContestMotherRussia        InlineButton = "MotherRussia"
	ContestChildProtectionDay  InlineButton = "ChildProtectionDay"
	ContestFire                InlineButton = "Fire"
	ContestTrafficLight        InlineButton = "TrafficLight"
	Agree                      InlineButton = "Согласен на обработку данных"
	Degree1                    InlineButton = "DEGREE1"
	Degree2                    InlineButton = "DEGREE2"
	Degree3                    InlineButton = "DEGREE3"
	Certificate                InlineButton = "Грамота"
	CertificateAndDiploma      InlineButton = "Грамота и диплом куратора"
	Diploma                    InlineButton = "Диплом куратора"
	FormatChoiceSingl          InlineButton = "цифрой"
	FormatChoiceGroup          InlineButton = "произвольный формат"
	PlaceDeliveryOfDocuments1  string       = "Электронная почта"
	PlaceDeliveryOfDocuments2  string       = "Телеграм"
	TableDB                    string       = "certificates"
	TimeshortForm              string       = "2006-01-02"
	Comma                      string       = ","
	Zero                       string       = "0"
	NoAge                      string       = "возраст не будет указан в грамоте/дипломе"
)

func (s InlineButton) String() string {
	switch s {
	case Agree:
		return "AGREE"
	case ContestTitmouse:
		return "Синичка невеличка и ee друзья"
	case ContestMather:
		return "Мама лучший друг"
	case ContestFather:
		return "Папа лучший друг"
	case ContestAutumn:
		return emoji.Sprint("Осень и ee дары 🍁")
	case ContestWinter:
		return "Зимушка-зима в гости к нам пришла"
	case ContestSnowflakes:
		return emoji.Sprint("Снежинки-балеринки :snowflake:")
	case ContestSnowman:
		return emoji.Sprint("Мой веселый снеговик :snowman:")
	case ContestSymbol:
		return "Символ года"
	case ContestHearts:
		return emoji.Sprint("Сердечки для любимых 💕")
	case ContestSecrets:
		return emoji.Sprint("Секреты новогодней ёлки 🎄")
	case ContestBirdsFeeding:
		return "Покормите птиц зимой"
	case ContestShrovetide:
		return emoji.Sprint("Широкая масленица 🌞")
	case ContestFable:
		return emoji.Sprint("В гостях у сказки :fairy:")
	case ContestDefendersFatherland:
		return emoji.Sprint("Защитники отечества ⚔️")
	case ContestSpring:
		return emoji.Sprint("Весна :blossom:")
	case ContestMarchEighth:
		return emoji.Sprint("8 Марта :bouquet:")
	case ContestEarth:
		return emoji.Sprint("Земля - наш общий дом 🌎")
	case ContestSpaceAdventures:
		return emoji.Sprint("Космические приключения :rocket:")
	case ContestFeatheredFriends:
		return emoji.Sprint("Пернатые друзья :owl:")
	case ContestTheatricalBackstage:
		return emoji.Sprint("Театральное закулисье 🎭")
	case ContestOurFriends:
		return "Наши друзья - Эколята"
	case ContestPrimroses:
		return emoji.Sprint("Первоцветы - лета первые шаги 🌺")
	case ContestVictoryDay:
		return "День Победы!"
	case ContestMyFamily:
		return "Моя семья - мое богатство"
	case ContestMotherRussia:
		return "Матушка Россия"
	case ContestChildProtectionDay:
		return "День защиты детей"
	case ContestFire:
		return emoji.Sprint("Не шути, дружок, с огнем 🔥")
	case ContestTrafficLight:
		return emoji.Sprint("В гостях у светофорика 🚦")
	case Degree1:
		return "1"
	case Degree2:
		return "2"
	case Degree3:
		return "3"
	case Diploma:
		return "diplomas"
	case Certificate:
		return "certificates"
	case CertificateAndDiploma:
		return "certificate and diploma"
	case FormatChoiceSingl:
		return "FormatChoiceSingl"
	case FormatChoiceGroup:
		return "FormatChoiceGroup"
	}

	return ""
}
