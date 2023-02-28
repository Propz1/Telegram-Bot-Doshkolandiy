package botstate

type BotState int64

const (
	// Since iota starts with 0, the first value
	// defined here will be the default
	Undefined BotState = iota // EnumIndex = 0
	Start                     // EnumIndex = 1
	Greeting
	Settings
	ShowSettings
	SelectProject
	CompleteApplication
	AskProject
	AskFNP
	AskFormatChoice
	AskAge
	AskNameInstitution
	AskLocality
	AskNamingUnit
	AskPublicationTitle
	AskFNPLeader
	AskEmail
	AskDocumentType
	AskPlaceDeliveryOfDocuments
	AskPhoto
	AskFile
	AskCheckData
	SelectCorrection
	AskFNPCorrection
	AskFormatChoiceCorrection
	AskAgeCorrection
	AskNameInstitutionCorrection
	AskLocalityCorrection
	AskNamingUnitCorrection
	AskPublicationTitleCorrection
	AskFNPLeaderCorrection
	AskEmailCorrection
	AskDocumentTypeCorrection
	AskPlaceDeliveryOfDocumentsCorrection
	AskPhotoCorrection
	AskFileCorrection
	AskRequisitionNumber
	AskDegree
	AskPublicationLink
	AskPublicationDate
	GetDiploma
)

func (s BotState) String() string {
	switch s {
	case Start:
		return "Start"
	}
	return "Undefined"
}

func (s BotState) EnumIndex() int64 {
	return int64(s)
}
