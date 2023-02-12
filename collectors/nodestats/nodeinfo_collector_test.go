package nodestats

import (
	"encoding/json"
	"io/ioutil"

	"github.com/kuskoman/logstash-exporter/fetcher/responses"
)

type mockClient struct{}

func (m *mockClient) GetNodeInfo() (*responses.NodeInfoResponse, error) {
	b, err := ioutil.ReadFile("node_info.json")
	if err != nil {
		return nil, err
	}

	var nodeInfo responses.NodeInfoResponse
	err = json.Unmarshal(b, &nodeInfo)
	if err != nil {
		return nil, err
	}

	return &nodeInfo, nil
}
