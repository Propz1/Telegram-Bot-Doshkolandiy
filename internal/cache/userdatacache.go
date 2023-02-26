package cache

import (
	"strconv"
	"sync"
	bs "telegrammBot/internal/botstate"
	"telegrammBot/internal/enumapplic"
)

type TempUsersIDCache struct {
	usersIDCache map[int64]struct{}
	mu           sync.Mutex
}

type userBotState map[int64]bs.BotState

type BotState struct {
	userBotState userBotState
	mu           sync.RWMutex
}

func NewTempUsersIDCache() *TempUsersIDCache {
	var t TempUsersIDCache
	t.usersIDCache = make(map[int64]struct{})
	return &t
}

func (t *TempUsersIDCache) Check(userID int64) bool {
	t.mu.Lock()
	_, found := t.usersIDCache[userID]
	t.mu.Unlock()
	return found
}

func (t *TempUsersIDCache) Add(userID int64) {
	t.mu.Lock()
	t.usersIDCache[userID] = struct{}{}
	t.mu.Unlock()
}

func (t *TempUsersIDCache) Delete(userID int64) {
	t.mu.Lock()
	if t.usersIDCache != nil {
		_, found := t.usersIDCache[userID]
		if found {
			delete(t.usersIDCache, userID)
		}
	}
	t.mu.Unlock()
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
	Age                    int
	GroupAge               string
	Group                  bool
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
	RequisitionNumber int64
	TableDB           string
	Diploma           bool
	DiplomaNumber     int64
	Degree            string
	PublicationLink   string
	PublicationDate   string
	UserID            int64
}

type ClosingRequisition struct {
	closingRequisitionCache map[int64]DataOfClosingRequisition
	//mu               sync.RWMutex
}

type DataPollingCache struct {
	userPollingCache map[int64]DataPolling
	//mu               sync.RWMutex
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

func (c *ClosingRequisition) Set(userID int64, enum enumapplic.ApplicEnum, text string) {

	var mu sync.RWMutex
	mu.Lock()

	st := c.closingRequisitionCache[userID]

	switch enum {

	case enumapplic.RequisitionNumber:
		num, _ := strconv.Atoi(text)
		st.RequisitionNumber = int64(num)
	case enumapplic.TableDB:
		st.TableDB = text
	case enumapplic.Diploma:
		st.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DiplomaNumber:
		num, _ := strconv.Atoi(text)
		st.DiplomaNumber = int64(num)
	case enumapplic.Degree:
		st.Degree = text
	case enumapplic.PublicationLink:
		st.PublicationLink = text
	case enumapplic.PublicationDate:
		st.PublicationDate = text
	case enumapplic.UserID:
		uid, _ := strconv.Atoi(text)
		st.UserID = int64(uid)
	}

	c.closingRequisitionCache[userID] = st
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

func (c *DataPollingCache) Set(userID int64, enum enumapplic.ApplicEnum, text string) {

	var mu sync.RWMutex
	mu.Lock()

	st := c.userPollingCache[userID]

	switch enum {

	case enumapplic.Contest:
		st.Contest = text
	case enumapplic.FNP:
		st.FNP = text
	case enumapplic.Age:
		age, _ := strconv.Atoi(text)
		st.Age = age
	case enumapplic.GroupAge:
		st.GroupAge = text
	case enumapplic.Group:
		st.Group = true
	case enumapplic.NotGroup:
		st.Group = false
	case enumapplic.NameInstitution:
		st.NameInstitution = text
	case enumapplic.Locality:
		st.Locality = text
	case enumapplic.NamingUnit:
		st.NamingUnit = text
	case enumapplic.PublicationTitle:
		st.PublicationTitle = text
	case enumapplic.FNPLeader:
		st.LeaderFNP = text
	case enumapplic.Email:
		st.Email = text
	case enumapplic.DocumentType:
		st.DocumentType = text
	case enumapplic.PlaceDeliveryOfDocuments:
		st.PlaceDeliveryDocuments = text
	case enumapplic.Photo:
		st.Photo = text
	case enumapplic.File:
		st.Files = append(st.Files, text)
	case enumapplic.RequisitionNumber:
		num, _ := strconv.Atoi(text)
		st.RequisitionNumber = int64(num)
	case enumapplic.RequisitionPDF:
		st.RequisitionPDFpath = text
	case enumapplic.TableDB:
		st.TableDB = text
	case enumapplic.Diploma:
		st.Diploma, _ = strconv.ParseBool(text)
	case enumapplic.DiplomaNumber:
		num, _ := strconv.Atoi(text)
		st.DiplomaNumber = int64(num)
	case enumapplic.Agree:
		st.Agree = true
	case enumapplic.PublicationLink:
		st.PublicationLink = text
	case enumapplic.PublicationDate:
		st.PublicationDate = text
	case enumapplic.Degree:
		num, _ := strconv.Atoi(text)
		st.Degree = num
	}

	c.userPollingCache[userID] = st
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
