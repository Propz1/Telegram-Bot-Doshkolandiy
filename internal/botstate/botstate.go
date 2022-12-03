package botstate

type BotState int64

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	Undefined BotState = iota //EnumIndex = 0
	START                     //EnumIndex = 1
	GREETING
	SETTINGS
	SHOW_SETTINGS
	SELECT_PROJECT
	COMPLETE_APPLICATION // Заполнить заявку/пройти опрос
	ASK_PROJECT
	ASK_FNP // ФИО
	ASK_AGE
	ASK_NAME_INSTITUTION  //Название учреждения (сокращенное)
	ASK_LOCALITY          //Населенный пункт
	ASK_NAMING_UNIT       //Номинация
	ASK_PUBLICATION_TITLE //Название работы
	ASK_FNP_LEADER        //ФИО руководителя
	ASK_EMAIL
	ASK_DOCUMENT_TYPE
	ASK_PLACE_DELIVERY_OF_DOCUMENTS
	ASK_CHECK_DATA
	SELECT_CORRECTION
	ASK_FNP_CORRECTION
	ASK_AGE_CORRECTION
	ASK_NAME_INSTITUTION_CORRECTION
	ASK_LOCALITY_CORRECTION 
	ASK_NAMING_UNIT_CORRECTION
	ASK_PUBLICATION_TITLE_CORRECTION
	ASK_FNP_LEADER_CORRECTION 
	ASK_EMAIL_CORRECTION 
	ASK_DOCUMENT_TYPE_CORRECTION 
	ASK_PLACE_DELIVERY_OF_DOCUMENTS_CORRECTION

	UNDEFINED
)

func (s BotState) String() string {
	switch s {

	case START:
		return "START"
	}
	return "Undefined"
}

func (s BotState) EnumIndex() int64 {
	return int64(s)
}
