package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	var compatibilityTable strings.Builder
	compatibilityTable.WriteString("| Logstash Version | Metric 1 | Metric 2 | Metric 3 |\n")
	compatibilityTable.WriteString("|------------------|----------|----------|----------|\n")

	for _, service := range services {
		if strings.HasPrefix(service.Spec.Name, "logstash") {
			version := strings.TrimPrefix(service.Spec.Name, "logstash_")
			compatibilityTable.WriteString("| " + version)

			resp, err := http.Get(fmt.Sprintf("http://%s:9600/_node/stats", service.Spec.Name))
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			// Check the compatibility of the metrics and add the results to the compatibility table
			// This is a simplified example and the actual implementation may vary
			if resp.StatusCode == http.StatusOK {
				compatibilityTable.WriteString(" | Yes | Yes | Yes |\n")
			} else {
				compatibilityTable.WriteString(" | No | No | No |\n")
			}
		}
	}

	err = ioutil.WriteFile("COMPATIBILITY.md", []byte(compatibilityTable.String()), 0644)
	if err != nil {
		panic(err)
	}
}