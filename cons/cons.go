package cons

type TablesDB string
type InlineButton string

const (
	TELEGRAMM_URL                             = "https://api.telegramm.org/bot"
	CERT_PAHT                                 = "./certs/cert.pem"
	KEY_PATH                                  = "./certs/key.pem"
	MaxLengMsg                   int          = 4000
	StyleTextMarkdown                         = "MarkdownV2"
	StyleTextCommon                           = ""
	StyleTextHTML                             = "HTML"
	PDF_PATH                                  = "./external/pdf/"
	PDF                                       = true
	FILE_PATH                                 = "./external/files/usersfiles"
	CONTEST_Titmouse             InlineButton = "Titmouse"
	CONTEST_Mather               InlineButton = "Mather"
	CONTEST_Father               InlineButton = "Father"
	CONTEST_Autumn               InlineButton = "Autumn"
	CONTEST_Winter               InlineButton = "Winter"
	CONTEST_Snowflakes           InlineButton = "Snowflakes"
	CONTEST_Snowman              InlineButton = "Snowman"
	CONTEST_Symbol               InlineButton = "Symbol"
	CONTEST_Heart                InlineButton = "Heart"
	CONTEST_Secrets              InlineButton = "Secrets"
	CONTEST_BirdsFeeding         InlineButton = "BirdsFeeding"
	CONTEST_Shrovetide           InlineButton = "Shrovetide"
	CONTEST_Fable                InlineButton = "Fable"
	AGREE                        InlineButton = "Согласен на обработку данных"
	DEGREE1                      InlineButton = "DEGREE1"
	DEGREE2                      InlineButton = "DEGREE2"
	DEGREE3                      InlineButton = "DEGREE3"
	CERTIFICATE                  InlineButton = "Грамота"
	CERTIFICATE_AND_DIPLOMA      InlineButton = "Грамота и диплом куратора"
	DIPLOMA                      InlineButton = "Диплом куратора"
	PLACE_DELIVERY_OF_DOCUMENTS1              = "Электронная почта"
	PLACE_DELIVERY_OF_DOCUMENTS2              = "Телеграмм"
	TableDB                                   = "certificates"
	TimeshortForm                             = "2006-02-01"
	Comma                                     = ","
)

func (s InlineButton) String() string {
	switch s {
	case AGREE:
		return "AGREE"
	case CONTEST_Titmouse:
		return "Синичка невеличка и ee друзья"
	case CONTEST_Mather:
		return "Мама лучший друг"
	case CONTEST_Father:
		return "Папа лучший друг"
	case CONTEST_Autumn:
		return "Осень и ee дары"
	case CONTEST_Winter:
		return "Зимушка-зима в гости к нам пришла"
	case CONTEST_Snowflakes:
		return "Снежинки-балеринки"
	case CONTEST_Snowman:
		return "Мой веселый снеговик"
	case CONTEST_Symbol:
		return "Символ года"
	case CONTEST_Heart:
		return "Сердечки для любимых"
	case CONTEST_Secrets:
		return "Секреты новогодней ёлки"
	case CONTEST_BirdsFeeding:
		return "Покормите птиц зимой"
	case CONTEST_Shrovetide:
		return "Широкая масленица"
	case CONTEST_Fable:
		return "В гостях у сказки"
	case DEGREE1:
		return "1"
	case DEGREE2:
		return "2"
	case DEGREE3:
		return "3"
	case DIPLOMA:
		return "diplomas"
	case CERTIFICATE:
		return "certificates"
	case CERTIFICATE_AND_DIPLOMA:
		return "certificate and diploma"
	}

	return ""
}
