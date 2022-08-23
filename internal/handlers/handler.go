package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"telegrammBot/cons"
	"telegrammBot/internal/models"
)

var (
	remainderList []models.Remainder
	mx            sync.Mutex
)

func RemainderHandler() (error, []models.Remainder) {

	remainderList = nil

	var username string = "Администратор"
	var passwd string = "Spbtechnoadmin311"

	client := &http.Client{}

	req, err := http.NewRequest("GET", cons.REMAINDER_REQUEST, nil)
	req.SetBasicAuth(os.Getenv("USERNAME_WEBSERVICE_1C"), os.Getenv("PASSWORD_WEBSERVICE_1C"))
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
		return fmt.Errorf("Bad GET request for remainder request:%W", err), nil
	}

	dataBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		log.Fatalln(err)
		return fmt.Errorf("Bad response for remainder request:%W", err), nil
	}

	if len(dataBody) != 0 {

		dataBody = bytes.TrimPrefix(dataBody, []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"

		mx.Lock()
		err = json.Unmarshal(dataBody, &remainderList)
		mx.Unlock()

		if err != nil {
			return err, nil
		}
	}

	return nil, remainderList
}
