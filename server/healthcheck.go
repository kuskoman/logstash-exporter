package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/kuskoman/logstash-exporter/config"
)

func getHealthCheck(logstashUrls []string) func(http.ResponseWriter, *http.Request) {
	client := &http.Client{}

	return func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		errorsChan := make(chan error, len(logstashUrls))

		for _, url := range logstashUrls {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(r.Context(), config.HttpTimeout)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				if err != nil {
					errorsChan <- fmt.Errorf("error creating request for %s: %v", url, err)
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					errorsChan <- fmt.Errorf("error making request to %s: %v", url, err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					errorsChan <- fmt.Errorf("%s returned status %d", url, resp.StatusCode)
					return
				}
			}(url)
		}

		wg.Wait()
		close(errorsChan)

		if len(errorsChan) > 0 {
			w.WriteHeader(http.StatusInternalServerError)
			for err := range errorsChan {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
