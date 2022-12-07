package cons

type TablesDB string

const (
	TELEGRAMM_URL                         = "https://api.telegramm.org/bot"
	CERT_PAHT                             = "./certs/cert.pem"
	KEY_PATH                              = "./certs/key.pem"
	MaxLengMsg                   int      = 4000
	StyleTextMarkdown                     = "MarkdownV2"
	StyleTextCommon                       = ""
	StyleTextHTML                         = "HTML"
	PDF_PATH                              = "./external/pdf/"
	PDF                                   = true
	CONTEST1                              = "Осень и ee дары"
	CONTEST2                              = "Синичка невеличка и ee друзья"
	CONTEST3                              = "Мама лучший друг"
	CONTEST4                              = "Методическая находка"
	DOCUMENT_TYPE1                        = "Грамота"
	DOCUMENT_TYPE2                        = "Диплом куратора"
	PLACE_DELIVERY_OF_DOCUMENTS1          = "Электронная почта"
	PLACE_DELIVERY_OF_DOCUMENTS2          = "Вконтакте"
	PLACE_DELIVERY_OF_DOCUMENTS3          = "Телеграмм"
	DIPLOMA                      TablesDB = "Диплом куратора"
	CERTIFICATE                  TablesDB = "Грамота"
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
