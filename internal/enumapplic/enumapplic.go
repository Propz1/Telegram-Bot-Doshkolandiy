package enumapplic

type ApplicEnum int64

const (
	// Since iota starts with 0, the first value
	// defined here will be the default
	Undefined ApplicEnum = iota // EnumIndex = 0
	Contest                     // EnumIndex = 1
	FNP                         // EnumIndex = 2
	Age
	NameInstitution
	Locality
	NamingUnit
	PublicationTitle
	FNPLeader
	Email
	DocumentType
	PlaceDeliveryOfDocuments
	Photo
	File
	Checking
	CancelCorrection
	RequisitionNumber
	RequisitionPDF
	DocumentPDF
	TableDB
	Diploma
	DiplomaNumber
	Agree
	Degree
	PublicationLink
	PublicationDate
	UserID
)

func (e ApplicEnum) String() string {
	switch e {
	case Contest:
		return "Участие в конкурсе"
	case FNP:
		return "ФИО / название группы участников"
	case Age:
		return "Возраст"
	case NameInstitution:
		return "Название учреждения (сокращенное)"
	case Locality:
		return "Населенный пункт"
	case NamingUnit:
		return "Название номинации"
	case PublicationTitle:
		return "Название работы"
	case FNPLeader:
		return "ФИО руководителя"
	case Email:
		return "Адрес электронной почты"
	case DocumentType:
		return "Тип документа"
	case PlaceDeliveryOfDocuments:
		return "Место получения документа"
	case Photo:
		return "Фотография работы"
	case File:
		return "Квитанция об оплате орг. взноса"
	case CancelCorrection:
		return "Отменить исправление"
	case RequisitionNumber:
		return "Номер заявки"
	case DiplomaNumber:
		return "Номер диплома куратора"
	case TableDB:
		return "Таблица базы данных"
	case Degree:
		return "Степень"
	case PublicationDate:
		return "Дата публикации работы"
	case PublicationLink:
		return "Ссылка на опубликованную работу"
	}
	return "Undefined"
}

func (e ApplicEnum) EnumIndex() int64 {
	return int64(e)
}
