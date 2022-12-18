package botcommand

type BotCommand int64

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	UNDEFINED BotCommand = iota //EnumIndex = 0
	SELECT_PROJECT
	CANCEL
	CANCEL_APPLICATION
	CONTINUE_DATA_POLLING
	DOWN
	SELECT_FNP_LEADER
	SELECT_DOCUMENT_TYPE
	SELECT_PLACE_DELIVERY_OF_DOCUMENTS
	CHECK_DATA
	COMPLETE_APPLICATION
	START_APPLICATION
	END_APPLICATION
	SETTINGS
	START
	SELECT_CORRECTION
	RECORD_TO_DB
	WAITING_FOR_ACCEPTANCE
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
	}

	return "Undefined"
}

func (c BotCommand) EnumIndex() int64 {
	return int64(c)
}
