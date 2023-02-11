package logstashclient

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestGetNodeInfo(t *testing.T) {
	t.Run("with valid response", func(t *testing.T) {
		fixtureContent, err := ioutil.ReadFile("../fixtures/node_stats.json")
		if err != nil {
			t.Fatalf("Error reading fixture file: %v", err)
		}
		handlerMock := &workingHandlerMock{fixture: string(fixtureContent)}
		client := NewClient(handlerMock)
		response, err := client.GetNodeInfo()
		if err != nil {
			t.Fatalf("Error getting node info: %v", err)
		}

		// we don't check every property, unmarschalling is tested in another test
		if (response.ID == "") || (response.Name == "") {
			t.Error("Expected response to be populated")
		}
	})

	t.Run("with invalid response", func(t *testing.T) {
		handlerMock := &failingHandlerMock{}
		client := NewClient(handlerMock)
		_, err := client.GetNodeInfo()
		if err == nil {
			t.Error("Expected error")
		}
	})
}

type workingHandlerMock struct {
	fixture string
}

func (h *workingHandlerMock) Get() (*http.Response, error) {
	reader := strings.NewReader(h.fixture)
	closer := ioutil.NopCloser(reader)
	response := &http.Response{Body: closer}
	return response, nil
}

type failingHandlerMock struct{}

func (h *failingHandlerMock) Get() (*http.Response, error) {
	return nil, errors.New("failed to get response")
}
