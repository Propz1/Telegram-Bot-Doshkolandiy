package cache

import (
	"strconv"
	"sync"
	bs "telegrammBot/internal/botstate"
	"telegrammBot/internal/enumapplic"
)

type userBotState map[int64]bs.BotState

type BotState struct {
	userBotState userBotState
	mu           sync.RWMutex
}

func NewCacheBotSt() BotState {
	return BotState{userBotState: make(userBotState, 0), mu: sync.RWMutex{}}
}

func (c *BotState) Get(userID int64) bs.BotState {
	c.mu.RLock()
	state, found := c.userBotState[userID]
	if !found {
		c.mu.RUnlock()
		return bs.Undefined
	}
	c.mu.RUnlock()
	return state
}

func (c *BotState) Set(userID int64, state bs.BotState) {
	c.mu.Lock()
	c.userBotState[userID] = state
	c.mu.Unlock()
}

func (c *BotState) Delete(userID int64) {
	if c.userBotState != nil {
		_, found := c.userBotState[userID]
		if found {
			delete(c.userBotState, userID)
		}
	}
}

type DataPolling struct {
	Contest                string
	FNP                    string
	Age                    string
	NameInstitution        string
	Locality               string
	NamingUnit             string
	PublicationTitle       string
	PublicationDate        string
	PublicationLink        string
	LeaderFNP              string
	Email                  string
	DocumentType           string
	PlaceDeliveryDocuments string
	RequisitionNumber      int64
	RequisitionPDFpath     string
	TableDB                string
	Diploma                bool
	DiplomaNumber          int64
	Agree                  bool
	Photo                  string
	Files                  []string
	Degree                 int
}

type DataOfClosingRequisition struct {
	//RequisitionNumber int64
	//TableDB           string
	//Diploma           bool
	//DiplomaNumber     int64
	//Degree            string
	//PublicationLink   string
	//PublicationDate   string
	UserID   int64
	UserData DataPolling
}

type ClosingRequisition struct {
	closingRequisitionCache map[int64]DataOfClosingRequisition
	// mu               sync.RWMutex
}

type DataPollingCache struct {
	userPollingCache map[int64]DataPolling
	// mu               sync.RWMutex
}

func NewCacheDataPolling() *DataPollingCache {
	var c DataPollingCache
	c.userPollingCache = make(map[int64]DataPolling)
	return &c
}

func NewCacheDataClosingRequisition() *ClosingRequisition {
	var c ClosingRequisition
	c.closingRequisitionCache = make(map[int64]DataOfClosingRequisition)
	return &c
}

func (c *ClosingRequisition) Get(userID int64) DataOfClosingRequisition {
	// c.mu.RLock()
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()
	st, found := c.closingRequisitionCache[userID]
	if !found {
		// c.mu.RUnlock()
		return st
	}
	// c.mu.RUnlock()

	return st
}

func (c *ClosingRequisition) Set(userID int64, enum enumapplic.ApplicEnum, text string) {
	var mu sync.RWMutex
	mu.Lock()

	storage := c.closingRequisitionCache[userID]

	switch enum {
	case enumapplic.UserID:
		uid, _ := strconv.Atoi(text)
		storage.UserID = int64(uid)
	case enumapplic.Contest:
		storage.UserData.Contest = text
	case enumapplic.FNP:
		storage.UserData.FNP = text
	case enumapplic.Age:
		storage.UserData.Age = text
	case enumapplic.NameInstitution:
		storage.UserData.NameInstitution = text
	case enumapplic.Locality:
		storage.UserData.Locality = text
	case enumapplic.NamingUnit:
		storage.UserData.NamingUnit = text
	case enumapplic.PublicationTitle:
		storage.UserData.PublicationTitle = text
	case enumapplic.FNPLeader:
		storage.UserData.LeaderFNP = text
	case enumapplic.Email:
		storage.UserData.Email = text
	case enumapplic.DocumentType:
		storage.UserData.DocumentType = text
	case enumapplic.PlaceDeliveryOfDocuments:
		storage.UserData.PlaceDeliveryDocuments = text
	case enumapplic.Photo:
		storage.UserData.Photo = text
	case enumapplic.File:
		storage.UserData.Files = append(storage.UserData.Files, text)
	case enumapplic.RequisitionNumber:
		num, _ := strconv.Atoi(text)
		storage.UserData.RequisitionNumber = int64(num)
	case enumapplic.RequisitionPDF:
		storage.UserData.RequisitionPDFpath = text
	case enumapplic.TableDB:
		storage.UserData.TableDB = text
	case enumapplic.Diploma:
		storage.UserData.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DiplomaNumber:
		num, _ := strconv.Atoi(text)
		storage.UserData.DiplomaNumber = int64(num)
	case enumapplic.Agree:
		storage.UserData.Agree = true
	case enumapplic.PublicationLink:
		storage.UserData.PublicationLink = text
	case enumapplic.PublicationDate:
		storage.UserData.PublicationDate = text
	case enumapplic.Degree:
		num, _ := strconv.Atoi(text)
		storage.UserData.Degree = num
	}

	c.closingRequisitionCache[userID] = storage
	mu.Unlock()
}

func (c *ClosingRequisition) Delete(userID int64) {
	var mu sync.RWMutex
	if c.closingRequisitionCache != nil {
		mu.Lock()
		_, found := c.closingRequisitionCache[userID]
		if found {
			delete(c.closingRequisitionCache, userID)
		}
		mu.Unlock()
	}
}

func (c *DataPollingCache) Get(userID int64) DataPolling {
	// c.mu.RLock()
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()
	st, found := c.userPollingCache[userID]
	if !found {
		// c.mu.RUnlock()
		return st
	}
	// c.mu.RUnlock()

	return st
}

func (c *DataPollingCache) Set(userID int64, enum enumapplic.ApplicEnum, text string) {
	var mu sync.RWMutex
	mu.Lock()

	storage := c.userPollingCache[userID]

	switch enum {
	case enumapplic.Contest:
		storage.Contest = text
	case enumapplic.FNP:
		storage.FNP = text
	case enumapplic.Age:
		storage.Age = text
	case enumapplic.NameInstitution:
		storage.NameInstitution = text
	case enumapplic.Locality:
		storage.Locality = text
	case enumapplic.NamingUnit:
		storage.NamingUnit = text
	case enumapplic.PublicationTitle:
		storage.PublicationTitle = text
	case enumapplic.FNPLeader:
		storage.LeaderFNP = text
	case enumapplic.Email:
		storage.Email = text
	case enumapplic.DocumentType:
		storage.DocumentType = text
	case enumapplic.PlaceDeliveryOfDocuments:
		storage.PlaceDeliveryDocuments = text
	case enumapplic.Photo:
		storage.Photo = text
	case enumapplic.File:
		storage.Files = append(storage.Files, text)
	case enumapplic.RequisitionNumber:
		num, _ := strconv.Atoi(text)
		storage.RequisitionNumber = int64(num)
	case enumapplic.RequisitionPDF:
		storage.RequisitionPDFpath = text
	case enumapplic.TableDB:
		storage.TableDB = text
	case enumapplic.Diploma:
		storage.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DiplomaNumber:
		num, _ := strconv.Atoi(text)
		storage.DiplomaNumber = int64(num)
	case enumapplic.Agree:
		storage.Agree = true
	case enumapplic.PublicationLink:
		storage.PublicationLink = text
	case enumapplic.PublicationDate:
		storage.PublicationDate = text
	case enumapplic.Degree:
		num, _ := strconv.Atoi(text)
		storage.Degree = num
	}

	c.userPollingCache[userID] = storage
	mu.Unlock()
}

func (c *DataPollingCache) Delete(userID int64) {
	var mu sync.RWMutex
	if c.userPollingCache != nil {
		mu.Lock()
		_, found := c.userPollingCache[userID]
		if found {
			delete(c.userPollingCache, userID)
		}
		mu.Unlock()
	}
}
