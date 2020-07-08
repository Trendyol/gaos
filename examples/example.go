package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type WorldTime struct {
	DateTime string `json:"datetime"`
}

var Port uint8 = 80

func HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		status := http.StatusOK

		header := w.Header()
		header.Set("Content-Type", "application/json")

		location := r.URL.Path[len("/search/"):]

		resp, err := http.Get(fmt.Sprintf("http://localhost:9080/api/timezone/Europe/%s", location))

		if err != nil {
			status = http.StatusNotAcceptable
			return
		}

		defer resp.Body.Close()

		body, readErr := ioutil.ReadAll(resp.Body)

		if readErr != nil {
			status = http.StatusUnprocessableEntity
		}

		worldTime := WorldTime{}
		jsonErr := json.Unmarshal(body, &worldTime)

		if jsonErr != nil {
			status = http.StatusExpectationFailed
		}

		if len(worldTime.DateTime) == 0 {
			status = http.StatusNotFound
		}

		w.WriteHeader(status)

		data := fmt.Sprintf(`{"location":"%s","status":"%d"}`, location, status)

		if status == http.StatusOK {
			data = fmt.Sprintf(`{"location":"%s","time":"%s","status":"%d"}`, location, worldTime.DateTime, status)
		}

		_, _ = w.Write([]byte(data))
	})
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", HTTPHandler())
	log.Printf("Server starting at :%d", Port)
	log.Fatalln(http.ListenAndServe(fmt.Sprintf(":%d", Port), mux))
}
