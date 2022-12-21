package enumapplic

type ApplicEnum int64

const (
	// since iota starts with 0, the first value
	// defined here will be the default
	UNDEFINED                   ApplicEnum = iota //EnumIndex = 0
	CONTEST                                       //EnumIndex = 1
	FNP                                           //EnumIndex = 2
	AGE                                           
	NAME_INSTITUTION                             
	LOCALITY                                      
	NAMING_UNIT                                   
	PUBLICATION_TITLE                             
	FNP_LEADER                                    
	EMAIL                                         
	DOCUMENT_TYPE                                
	PLACE_DELIVERY_OF_DOCUMENTS                   
	PHOTO                                         
	FILE                                          
	CHECKING                                      
	CANSEL_CORRECTION                             
	REQUISITION_NUMBER                           
	REQUISITION_PDF                               
	TableDB                                       
	Agree                                         
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
		return "Место получения документа"
	case PHOTO:
		return "Фотография работы"
	case FILE:
		return "Квитанция об оплате"
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
