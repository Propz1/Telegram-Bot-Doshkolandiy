package botcommand

type BotCommand int64

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	UNDEFINED BotCommand = iota //EnumIndex = 0
	SELECT_PROJECT
	CANCEL
	CANCEL_APPLICATION
	CANCEL_CLOSE_REQUISITION
	CONTINUE
	CONTINUE_DATA_POLLING
	FURTHER
	DOWN
	SELECT_FNP_LEADER
	SELECT_DOCUMENT_TYPE
	SELECT_PLACE_DELIVERY_OF_DOCUMENTS
	CHECK_DATA
	CHECK_PDF_FILES
	COMPLETE_APPLICATION
	START_APPLICATION
	END_APPLICATION
	SETTINGS
	START
	SELECT_CORRECTION
	CONFIRM
	RECORD_TO_DB
	WAITING_FOR_ACCEPTANCE
	CLOSE_REQUISITION_START
	SELECT_DEGREE
	GET_PUBLICATION_LINK
	GET_PUBLICATION_DATE
	CLOSE_REQUISITION_END
	GET_DIPLOMA
	ACCESS_DENIED
	SEND_PDF_FILES
)

func (c BotCommand) String() string {
	switch c {

	case COMPLETE_APPLICATION:
		return "Заполнить заявку"
	case SELECT_PROJECT:
		return "Продолжить"
	case CANCEL:
		return "Отмена"
	case CANCEL_APPLICATION:
		return "Отменить заявку"
	case DOWN:
		return "Далее"
	case SELECT_CORRECTION:
		return "Исправить"
	case CLOSE_REQUISITION_START:
		return "Закрыть заявку"
	case SETTINGS:
		return "Настройки"
	case CONFIRM:
		return "Подтвердить"
	case CONTINUE:
		return "Продолжить"
	case FURTHER:
		return "Далее"
	case GET_DIPLOMA:
		return "Получить диплом"
	case SEND_PDF_FILES:
		return "Подтверждаю закрытие"
	case CANCEL_CLOSE_REQUISITION:
		return "Отменяю закрытие"
	case START:
		return "/start"
	}

	return "Undefined"
}

func (c BotCommand) EnumIndex() int64 {
	return int64(c)
}
