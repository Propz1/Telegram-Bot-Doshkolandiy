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
	CONTEST1                                  = "Осень и ee дары"
	CONTEST2                                  = "Синичка невеличка и ee друзья"
	CONTEST3                                  = "Мама лучший друг"
	CONTEST4                                  = "Методическая находка"
	DOCUMENT_TYPE1                            = "Грамота"
	DOCUMENT_TYPE2                            = "Диплом куратора"
	PLACE_DELIVERY_OF_DOCUMENTS1              = "Электронная почта"
	PLACE_DELIVERY_OF_DOCUMENTS2              = "Телеграмм"
	DIPLOMA                      TablesDB     = "Диплом куратора"
	CERTIFICATE                  TablesDB     = "Грамота"
	AGREE                        InlineButton = "Согласен на обработку данных"
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
	}

	return ""
}
