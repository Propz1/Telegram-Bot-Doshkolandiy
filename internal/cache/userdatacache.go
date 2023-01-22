package cache

import (
	"strconv"
	"sync"
	bs "telegrammBot/internal/botstate"
	"telegrammBot/internal/enumapplic"
)

type userBotState map[int64]bs.BotState

type CacheBotSt struct {
	userBotState userBotState
	mu           sync.RWMutex
}

func NewCacheBotSt() CacheBotSt {
	return CacheBotSt{userBotState: make(userBotState, 0), mu: sync.RWMutex{}}
}

func (c *CacheBotSt) Get(userID int64) bs.BotState {
	c.mu.RLock()
	state, found := c.userBotState[userID]
	if !found {
		c.mu.RUnlock()
		return bs.Undefined
	}
	c.mu.RUnlock()
	return state
}

func (c *CacheBotSt) Set(userID int64, state bs.BotState) {
	c.mu.Lock()
	c.userBotState[userID] = state
	c.mu.Unlock()
}

func (c *CacheBotSt) Delete(userID int64) {
	if c.userBotState != nil {
		_, found := c.userBotState[userID]
		if found {
			delete(c.userBotState, userID)
		}
	}
}

type dataPolling struct {
	Contest                string
	FNP                    string
	Age                    int
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

type dataClosingRequisition struct {
	RequisitionNumber int64
	TableDB           string
	Diploma           bool
	DiplomaNumber     int64
	Degree            string
	PublicationLink   string
	PublicationDate   string
	UserID            int64
}

type CacheDataClosingRequisition struct {
	closingRequisitionCache map[int64]dataClosingRequisition
	//mu               sync.RWMutex
}

type CacheDataPolling struct {
	userPollingCache map[int64]dataPolling
	//mu               sync.RWMutex
}

func NewCacheDataPolling() *CacheDataPolling {
	var c CacheDataPolling
	c.userPollingCache = make(map[int64]dataPolling)
	return &c
}

func NewCacheDataClosingRequisition() *CacheDataClosingRequisition {
	var c CacheDataClosingRequisition
	c.closingRequisitionCache = make(map[int64]dataClosingRequisition)
	return &c
}

func (c *CacheDataClosingRequisition) Get(userID int64) dataClosingRequisition {
	//c.mu.RLock()
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()
	st, found := c.closingRequisitionCache[userID]
	if !found {
		//c.mu.RUnlock()
		return st
	}
	//c.mu.RUnlock()

	return st
}

func (c *CacheDataClosingRequisition) Set(userID int64, enum enumapplic.ApplicEnum, text string) {

	var mu sync.RWMutex
	mu.Lock()

	st := c.closingRequisitionCache[userID]

	switch enum {

	case enumapplic.REQUISITION_NUMBER:
		num, _ := strconv.Atoi(text)
		st.RequisitionNumber = int64(num)
	case enumapplic.TableDB:
		st.TableDB = text
	case enumapplic.DIPLOMA:
		st.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DIPLOMA_NUMBER:
		num, _ := strconv.Atoi(text)
		st.DiplomaNumber = int64(num)
	case enumapplic.DEGREE:
		st.Degree = text
	case enumapplic.PUBLICATION_LINK:
		st.PublicationLink = text
	case enumapplic.PUBLICATION_DATE:
		st.PublicationDate = text
	case enumapplic.USER_ID:
		u_id, _ := strconv.Atoi(text)
		st.UserID = int64(u_id)
	}

	c.closingRequisitionCache[userID] = st
	mu.Unlock()
}

func (c *CacheDataClosingRequisition) Delete(userID int64) {
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

func (c *CacheDataPolling) Get(userID int64) dataPolling {
	//c.mu.RLock()
	var mu sync.RWMutex
	mu.RLock()
	defer mu.RUnlock()
	st, found := c.userPollingCache[userID]
	if !found {
		//c.mu.RUnlock()
		return st
	}
	//c.mu.RUnlock()

	return st
}

func (c *CacheDataPolling) Set(userID int64, enum enumapplic.ApplicEnum, text string) {

	var mu sync.RWMutex
	mu.Lock()

	st := c.userPollingCache[userID]

	switch enum {

	case enumapplic.CONTEST:
		st.Contest = text
	case enumapplic.FNP:
		st.FNP = text
	case enumapplic.AGE:
		age, _ := strconv.Atoi(text)
		st.Age = age
	case enumapplic.NAME_INSTITUTION:
		st.NameInstitution = text
	case enumapplic.LOCALITY:
		st.Locality = text
	case enumapplic.NAMING_UNIT:
		st.NamingUnit = text
	case enumapplic.PUBLICATION_TITLE:
		st.PublicationTitle = text
	case enumapplic.FNP_LEADER:
		st.LeaderFNP = text
	case enumapplic.EMAIL:
		st.Email = text
	case enumapplic.DOCUMENT_TYPE:
		st.DocumentType = text
	case enumapplic.PLACE_DELIVERY_OF_DOCUMENTS:
		st.PlaceDeliveryDocuments = text
	case enumapplic.PHOTO:
		st.Photo = text
	case enumapplic.FILE:
		st.Files = append(st.Files, text)
	case enumapplic.REQUISITION_NUMBER:
		num, _ := strconv.Atoi(text)
		st.RequisitionNumber = int64(num)
	case enumapplic.REQUISITION_PDF:
		st.RequisitionPDFpath = text
	case enumapplic.TableDB:
		st.TableDB = text
	case enumapplic.DIPLOMA:
		st.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DIPLOMA_NUMBER:
		num, _ := strconv.Atoi(text)
		st.DiplomaNumber = int64(num)
	case enumapplic.AGREE:
		st.Agree = true
	case enumapplic.PUBLICATION_LINK:
		st.PublicationLink = text
	case enumapplic.PUBLICATION_DATE:
		st.PublicationDate = text
	case enumapplic.DEGREE:
		num, _ := strconv.Atoi(text)
		st.Degree = num
	}

	c.userPollingCache[userID] = st
	mu.Unlock()

}

func (c *CacheDataPolling) Delete(userID int64) {
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
