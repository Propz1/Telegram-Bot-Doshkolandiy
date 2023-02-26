package cons

//type TablesDB string
type InlineButton string

const (
	TelegrammURL                            = "https://api.telegramm.org/bot"
	CertPaht                                = "./certs/cert.pem"
	KeyPath                                 = "./certs/key.pem"
	MaxLengMsg                 int          = 4000
	StyleTextMarkdown                       = "MarkdownV2"
	StyleTextCommon                         = ""
	StyleTextHTML                           = "HTML"
	PDFPath                                 = "./external/pdf/"
	PDF                                     = true
	FilePath                                = "./external/files/usersfiles"
	ContestTitmouse            InlineButton = "Titmouse"
	ContestMather              InlineButton = "Mather"
	ContestFather              InlineButton = "Father"
	ContestAutumn              InlineButton = "Autumn"
	ContestWinter              InlineButton = "Winter"
	ContestSnowflakes          InlineButton = "Snowflakes"
	ContestSnowman             InlineButton = "Snowman"
	ContestSymbol              InlineButton = "Symbol"
	ContestHeart               InlineButton = "Heart"
	ContestSecrets             InlineButton = "Secrets"
	ContestBirdsFeeding        InlineButton = "BirdsFeeding"
	ContestShrovetide          InlineButton = "Shrovetide"
	ContestFable               InlineButton = "Fable"
	ContestDefendersFatherland InlineButton = "DefendersFatherland"
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
	TableDB                                 = "certificates"
	TimeshortForm                           = "2006-02-01"
	Comma                                   = ","
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
		return "Осень и ee дары"
	case ContestWinter:
		return "Зимушка-зима в гости к нам пришла"
	case ContestSnowflakes:
		return "Снежинки-балеринки"
	case ContestSnowman:
		return "Мой веселый снеговик"
	case ContestSymbol:
		return "Символ года"
	case ContestHeart:
		return "Сердечки для любимых"
	case ContestSecrets:
		return "Секреты новогодней ёлки"
	case ContestBirdsFeeding:
		return "Покормите птиц зимой"
	case ContestShrovetide:
		return "Широкая масленица"
	case ContestFable:
		return "В гостях у сказки"
	case ContestDefendersFatherland:
		return "Защитники отечества"
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
