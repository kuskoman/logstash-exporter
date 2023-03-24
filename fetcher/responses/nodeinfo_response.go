package responses

// NodeInfoResponse is the response from the "/" endpoint of the Logstash API
type NodeInfoResponse struct {
	Host        string `json:"host"`
	Version     string `json:"version"`
	HTTPAddress string `json:"http_address"`
	ID          string `json:"id"`
	Name        string `json:"name"`
	EphemeralID string `json:"ephemeral_id"`
	Status      string `json:"status"`
	Snapshot    bool   `json:"snapshot"`
	Pipeline    struct {
		Workers    int `json:"workers"`
		BatchSize  int `json:"batch_size"`
		BatchDelay int `json:"batch_delay"`
	} `json:"pipeline"`
	BuildDate     string `json:"build_date"`
	BuildSHA      string `json:"build_sha"`
	BuildSnapshot bool   `json:"build_snapshot"`
}
