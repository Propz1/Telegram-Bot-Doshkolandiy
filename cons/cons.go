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
	DEGREE1                       InlineButton = "DEGREE1"
	DEGREE2                       InlineButton = "DEGREE2"
	DEGREE3                       InlineButton = "DEGREE3"
	DOCUMENT_TYPE1                            = "Грамота"
	DOCUMENT_TYPE2                            = "Диплом куратора"
	DOCUMENT_TYPE3                            = "Грамота + Диплом куратора"
	PLACE_DELIVERY_OF_DOCUMENTS1              = "Электронная почта"
	PLACE_DELIVERY_OF_DOCUMENTS2              = "Телеграмм"
	DIPLOMA                      TablesDB     = "Диплом куратора"
	CERTIFICATE                  TablesDB     = "Грамота"
)

func (s TablesDB) String() string {
	switch s {
	case DIPLOMA:
		return "diplomas"
	case CERTIFICATE:
		return "certificates"
	}

	return ""
}

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
	}

	return ""
}
