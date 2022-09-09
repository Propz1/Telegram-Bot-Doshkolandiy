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
	remainderList      models.ArrayRemainder
	mx                 sync.Mutex
	listWarehouses     models.Warehouses
	warehouseRemainder models.WarehouseRemainder
)

func MovementsHandler() (error, models.ArrayRemainder) {

	remainderList = nil

	client := &http.Client{}

	req, err := http.NewRequest("GET", cons.MOVEMENTS_REQUEST, nil)
	req.SetBasicAuth(os.Getenv("USERNAME_WEBSERVICE_1C"), os.Getenv("PASSWORD_WEBSERVICE_1C"))
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
		return fmt.Errorf("Bad GET request for MOVEMENTS request:%W", err), nil
	}

	dataBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if err != nil {
		log.Fatalln(err)
		return fmt.Errorf("Bad response for MOVEMENTS request:%W", err), nil
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

func RemainderHandler(warehouse string) (models.WarehouseRemainder, error) {

	//remainderInformation = nil

	//client := &http.Client{Timeout: 10 * time.Second}
	client := &http.Client{}

	message := make(map[string]string)
	message["Command"] = "RemainderRequest"
	message["Склад"] = warehouse

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		log.Fatalln(err)
	}

	req, err := http.NewRequest("POST", cons.REMAINDER_REQUEST, bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
		return models.WarehouseRemainder{}, fmt.Errorf("Bad POST request for remainder request:%W", err)
	}
	req.SetBasicAuth(os.Getenv("USERNAME_WEBSERVICE_1C"), os.Getenv("PASSWORD_WEBSERVICE_1C"))
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
		return models.WarehouseRemainder{}, fmt.Errorf("Bad GET/POST request for remainder request:%W", err)
	}

	dataBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	//fmt.Printf("dataBody is %v\n", string(dataBody))

	if err != nil {
		log.Fatalln(err)
		return models.WarehouseRemainder{}, fmt.Errorf("Bad response for remainder request:%W", err)
	}

	if len(dataBody) != 0 {

		dataBody = bytes.TrimPrefix(dataBody, []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"

		mx.Lock()
		err = json.Unmarshal(dataBody, &warehouseRemainder)
		mx.Unlock()

		if err != nil {
			return models.WarehouseRemainder{}, err
		}

	}

	return warehouseRemainder, nil
}

func WarehousesHandler() (models.Warehouses, error) {

	//client := &http.Client{Timeout: 10 * time.Second}
	client := &http.Client{}

	message := make(map[string]string, 1)
	message["Command"] = "GetWarehouses"

	bytesRepresentation, err := json.Marshal(message)
	if err != nil {
		log.Fatalln(err)
	}

	req, err := http.NewRequest("POST", cons.REMAINDER_REQUEST, bytes.NewBuffer(bytesRepresentation))
	if err != nil {
		log.Fatalln(err)
		return models.Warehouses{}, fmt.Errorf("Bad GET/POST request for remainder request:%W", err)
	}
	req.SetBasicAuth(os.Getenv("USERNAME_WEBSERVICE_1C"), os.Getenv("PASSWORD_WEBSERVICE_1C"))
	resp, err := client.Do(req)

	if err != nil {
		log.Fatalln(err)
		return models.Warehouses{}, fmt.Errorf("Bad GET/POST request for remainder request:%W", err)
	}

	dataBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	fmt.Printf("dataBody is %v\n", string(dataBody))

	if err != nil {
		log.Fatalln(err)
		return models.Warehouses{}, fmt.Errorf("Bad response for remainder request:%W", err)
	}

	if len(dataBody) != 0 {

		dataBody = bytes.TrimPrefix(dataBody, []byte("\xef\xbb\xbf")) //For error deletion of type "invalid character 'ï' looking for beginning of value"

		mx.Lock()
		err = json.Unmarshal(dataBody, &listWarehouses)
		mx.Unlock()

		if err != nil {
			return models.Warehouses{}, err
		}

	}

	return listWarehouses, nil
}
