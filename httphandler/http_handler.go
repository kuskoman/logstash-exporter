package httphandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type DefaultHTTPHandler struct {
	Endpoint string
}

func GetDefaultHTTPHandler(endpoint string) HTTPHandler {
	return &DefaultHTTPHandler{Endpoint: endpoint}
}

func (h *DefaultHTTPHandler) Get(path string) (*http.Response, error) {
	url := h.Endpoint + path
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type HTTPHandler interface {
	Get(string) (*http.Response, error)
}

func GetMetrics(h HTTPHandler, path string, target interface{}) error {
	response, err := h.Get("")
	if err != nil {
		return errors.New("Cannot get metrics from Logstash: " + err.Error())
	}

	defer func() {
		err = response.Body.Close()
		if err != nil {
			log.Printf("Cannot close response body: %v", err)
		}
	}()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		log.Printf("Cannot parse Logstash response json: %s", err)
	}

	return nil
}
