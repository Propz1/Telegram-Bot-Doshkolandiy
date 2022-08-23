package webserver

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"telegrammBot/internal/handlers"
	"telegrammBot/internal/types"

	"github.com/gorilla/mux"
)

type handler struct {
}



func (h *handler) Register(router *mux.Router) {
	// router.HandleFunc("/bad", BadHandler).Methods("GET")
	// router.HandleFunc("/name/{PARAM}", NameHandler).Methods("GET")
	router.HandleFunc("/", MessageHandler).Methods("GET")
	router.HandleFunc("/", MessageHandler).Methods("POST")
	// router.HandleFunc("/headers", SumHandler).Methods("POST")
}

// func BadHandler(w http.ResponseWriter, r *http.Request) {
// 	w.WriteHeader(http.StatusInternalServerError)
// }

// func NameHandler(w http.ResponseWriter, r *http.Request) {
// 	vars := mux.Vars(r)
// 	w.WriteHeader(http.StatusOK)
// 	fmt.Fprintf(w, "Hello, %v!", vars["PARAM"])
// }

func MessageHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatalln(err)
	}
	var botText types.BotMessage

	err = json.Unmarshal(body, &botText)

	if err != nil {
		log.Fatalln(err)
	}

	fmt.Fprintf(w, "I got message:\n%v", string(body))

	userName := botText.Message.From.Username
	chatUser := botText.Message.From.Id
	chatGroup := botText.Message.Chat.Id
	messageID := botText.Message.Message_id
	botCommand := strings.Split(botText.Message.Text, "@")[0]
	commandText := strings.Split(botText.Message.Text, " ")

	fmt.Println(userName, chatUser, chatGroup, messageID, botCommand, commandText)

}

// func SumHandler(w http.ResponseWriter, r *http.Request) {

// 	httpStatus = http.StatusOK

// 	a := r.Header.Get("a")
// 	b := r.Header.Get("b")

// 	sum, err := Sum(a, b)

// 	if err != nil {
// 		httpStatus = http.StatusBadRequest
// 		log.Fatalln(err)
// 	}

// 	body, err := ioutil.ReadAll(r.Body)
// 	defer r.Body.Close()

// 	if err != nil {
// 		httpStatus = http.StatusBadRequest
// 		log.Fatalln(err)
// 	}

// 	w.Header().Set("a+b", sum)
// 	w.WriteHeader(httpStatus)

// 	if len(body) == 0 {
// 		fmt.Fprintf(w, "Empty body")
// 	}

// }

// func Sum(a, b string) (string, error) {

// 	x1, err := strconv.Atoi(a)

// 	if err != nil {
// 		log.Fatalln(err)
// 		return "", err
// 	}

// 	x2, err := strconv.Atoi(b)

// 	if err != nil {
// 		log.Fatalln(err)
// 		return "", err
// 	}

// 	sum := x1 + x2

// 	return strconv.Itoa(sum), nil
// }
