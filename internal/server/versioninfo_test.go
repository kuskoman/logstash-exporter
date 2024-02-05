package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kuskoman/logstash-exporter/pkg/config"
)

type writerMock struct {
	writes [][]byte
}

func (w *writerMock) Header() http.Header {
	return http.Header{}
}

func (w *writerMock) Write(input []byte) (int, error) {
	w.writes = append(w.writes, input)

	if len(w.writes) < 2 {
		return 0, io.EOF
	}

	return 0, nil
}

func (w *writerMock) WriteHeader(statusCode int) {}

func TestHandleVersionInfo(t *testing.T) {
	t.Run("normal case", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(getVersionInfoHandler(config.GetVersionInfo())))
		defer ts.Close()

		resp, err := http.Get(ts.URL)
		if err != nil {
			t.Fatalf("failed to make a request to the test server: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read the response body: %v", err)
		}

		var versionInfo config.VersionInfo
		err = json.Unmarshal(body, &versionInfo)
		if err != nil {
			t.Fatalf("failed to decode JSON: %v", err)
		}

		expectedVersionInfo := config.GetVersionInfo()
		if versionInfo != *expectedVersionInfo {
			t.Errorf("expected version info: %+v, but got: %+v", *expectedVersionInfo, versionInfo)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code %d, but got: %d", http.StatusOK, resp.StatusCode)
		}

		if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type header to be 'application/json', but got: %s", contentType)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		versionInfo := &config.VersionInfo{
			Version:   "version",
			GitCommit: "git commit",
			GoVersion: "go version",
			BuildArch: "build arch",
			BuildOS:   "build os",
			BuildDate: "build date",
		}
		handler := getVersionInfoHandler(versionInfo)

		w := &writerMock{}

		handler(w, nil)

		if len(w.writes) != 2 {
			t.Errorf("expected 2 writes, but got: %d", len(w.writes))
		}

		firstWrite := string(w.writes[0])
		expectedFirstWrite := `{"Version":"version","SemanticVersion":"","GitCommit":"git commit","GoVersion":"go version","BuildArch":"build arch","BuildOS":"build os","BuildDate":"build date"}`
		expectedFirstWrite = expectedFirstWrite + "\n"
		if firstWrite != expectedFirstWrite {
			t.Errorf("expected first write to be %s, but got: %s", expectedFirstWrite, firstWrite)
		}

		secondWrite := string(w.writes[1])
		expectedSecondWrite := "EOF\n"
		if secondWrite != expectedSecondWrite {
			t.Errorf("expected second write to be %s, but got: %s", expectedSecondWrite, secondWrite)
		}
	})
}
