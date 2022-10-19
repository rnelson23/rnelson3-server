package main

import (
	"encoding/json"
	"github.com/JamesPEarly/loggly"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"net/http"
	"time"
)

type Time struct {
	SystemTime string `json:"time"`
}

type Log struct {
	Method      string `json:"method"`
	SourceIP    string `json:"ip"`
	RequestPath string `json:"path"`
	StatusCode  int    `json:"status"`
}

func main() {
	router := mux.NewRouter()
	_ = godotenv.Load()

	router.HandleFunc("/rnelson3/status", handler)
	router.PathPrefix("/rnelson3/").HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusNotFound)
		log(req, http.StatusNotFound)
	})

	_ = loggly.New("reddit-server").EchoSend("info", "Ready!")
	_ = http.ListenAndServe(":8080", router)
}

func handler(res http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		res.WriteHeader(http.StatusMethodNotAllowed)
		log(req, http.StatusMethodNotAllowed)
		return
	}

	res.Header().Add("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)

	data, _ := json.MarshalIndent(Time{SystemTime: time.Now().String()}, "", "    ")
	_, _ = res.Write(data)
	log(req, http.StatusOK)
}

func log(req *http.Request, statusCode int) {
	bytes, _ := json.MarshalIndent(Log{
		Method:      req.Method,
		SourceIP:    req.RemoteAddr,
		RequestPath: req.URL.Path,
		StatusCode:  statusCode,
	}, "", "    ")

	_ = loggly.New("reddit-server").Send("info", string(bytes))
}
