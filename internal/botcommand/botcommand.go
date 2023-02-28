package botcommand

type BotCommand int64

const (
	// Since iota starts with 0, the first value
	// defined here will be the default
	Undefined BotCommand = iota // EnumIndex = 0
	SelectProject
	Cancel
	CancelApplication
	CancelCloseRequisition
	Continue
	ContinueDataPolling
	Further
	Down
	SelectFNPLeader
	SelectDocumentType
	SelectPlaceDeliveryOfDocuments
	CheckData
	CheckDataPause
	CheckPDFFiles
	CompleteApplication
	EndApplication
	Settings
	Start
	SelectCorrection
	Confirm
	RecordToDB
	WaitingForAcceptance
	CloseRequisitionStart
	SelectDegree
	GetPublicationLink
	GetPublicationDate
	CloseRequisitionEnd
	GetDiploma
	AccessDenied
	SendPDFFiles
	FormatChoice
)

func (c BotCommand) String() string {
	switch c {
	case CompleteApplication:
		return "Заполнить заявку"
	case SelectProject:
		return "Продолжить"
	case Cancel:
		return "Отмена"
	case CancelApplication:
		return "Отменить заявку"
	case Down:
		return "Далее"
	case SelectCorrection:
		return "Исправить"
	case CloseRequisitionStart:
		return "Закрыть заявку"
	case Settings:
		return "Настройки"
	case Confirm:
		return "Подтвердить"
	case Continue:
		return "Продолжить"
	case Further:
		return "Далее"
	case GetDiploma:
		return "Получить диплом"
	case SendPDFFiles:
		return "Подтверждаю закрытие"
	case CancelCloseRequisition:
		return "Отменяю закрытие"
	case Start:
		return "/start"
	}

	return "Undefined"
}

func (c BotCommand) EnumIndex() int64 {
	return int64(c)
}
