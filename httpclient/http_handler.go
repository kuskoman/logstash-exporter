package httpclient

import (
	"encoding/json"
	"log"
	"net/http"
)

type DefaultHTTPHandler struct {
	Endpoint string
}

func GetDefaultHTTPHandler(endpoint string) HTTPHandler {
	return &DefaultHTTPHandler{Endpoint: endpoint}
}

func (h *DefaultHTTPHandler) Get() (*http.Response, error) {
	response, err := http.Get(h.Endpoint)
	if err != nil {
		return nil, err
	}

	return response, nil
}

type HTTPHandler interface {
	Get() (*http.Response, error)
}

func GetMetrics(h HTTPHandler, target interface{}) error {
	response, err := h.Get()
	if err != nil {
		log.Printf("Cannot retrieve metrics: %s", err)
		return nil
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
