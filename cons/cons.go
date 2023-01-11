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
	CONTEST_Autumn               InlineButton = "Autumn"
	CONTEST_Titmouse             InlineButton = "Titmouse"
	CONTEST_Mather               InlineButton = "Mather"
	CONTEST_Find                 InlineButton = "Find"
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
	TimeshortForm                             = "2006-01-02"
)

func (s InlineButton) String() string {
	switch s {
	case AGREE:
		return "AGREE"
	case CONTEST_Titmouse:
		return "Синичка невеличка и ee друзья"
	case CONTEST_Mather:
		return "Мама лучший друг"
	case CONTEST_Find:
		return "Методическая находка"
	case CONTEST_Autumn:
		return "Осень и ee дары"
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
