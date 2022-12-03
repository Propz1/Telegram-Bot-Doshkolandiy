package enumapplic

type ApplicEnum int64

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	UNDEFINED                   ApplicEnum = iota //EnumIndex = 0
	CONTEST                                       //Конкурс                     //EnumIndex = 1
	FNP                                           //ФИО                        //EnumIndex = 2
	AGE                                           //Возраст                    ////EnumIndex = 3
	NAME_INSTITUTION                              //Название учреждения (сокращенное)
	LOCALITY                                      //Населенный пункт
	NAMING_UNIT                                   //Номинация
	PUBLICATION_TITLE                             //Название работы
	FNP_LEADER                                    //ФИО руководителя
	EMAIL                                         //e-mail
	DOCUMENT_TYPE                                 //Диплом куратора или грамота
	PLACE_DELIVERY_OF_DOCUMENTS                   //Место получения диплома/грамоты
	CHECKING                                      //Проверка
	CANSEL_CORRECTION                             //Отменить исправление
	REQUISITION_NUMBER                            //Номер заявки
)

func (e ApplicEnum) String() string {

	switch e {

	case CONTEST:
		return "Участие в конкурсе"
	case FNP:
		return "ФИО"
	case AGE:
		return "Возраст"
	case NAME_INSTITUTION:
		return "Название учреждения (сокращенное)"
	case LOCALITY:
		return "Населенный пункт"
	case NAMING_UNIT:
		return "Название номинации"
	case PUBLICATION_TITLE:
		return "Название работы"
	case FNP_LEADER:
		return "ФИО руководителя"
	case EMAIL:
		return "Адрес электронной почты"
	case DOCUMENT_TYPE:
		return "Тип документа"
	case PLACE_DELIVERY_OF_DOCUMENTS:
		return "Место получения диплома/грамоты"
	case CANSEL_CORRECTION:
		return "Отменить исправление"
	case REQUISITION_NUMBER:
		return "Номер заявки"
	}
	return "Undefined"
}

func (e ApplicEnum) EnumIndex() int64 {
	return int64(e)
}
