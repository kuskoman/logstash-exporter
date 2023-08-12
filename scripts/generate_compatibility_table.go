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

	logstashVersions := strings.Split(os.Getenv("LOGSTASH_VERSIONS"), ",")

	services, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		panic(err)
	}

	var compatibilityTable strings.Builder
	compatibilityTable.WriteString("| Logstash Version | Metric 1 | Metric 2 | Metric 3 |\n")
	compatibilityTable.WriteString("|------------------|----------|----------|----------|\n")

 	for _, service := range services {
 		for _, version := range logstashVersions {
 			if strings.HasPrefix(service.Spec.Name, "logstash_"+version) {
 				compatibilityTable.WriteString("| " + version)

   			resp, err := http.Get(fmt.Sprintf("http://logstash_%s:9600/_node/stats", version))
			if err != nil {
				fmt.Printf("Error getting metrics from Logstash: %v\n", err)
				continue
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response body: %v\n", err)
				continue
			}

			var metrics Metrics
			err = json.Unmarshal(body, &metrics)
			if err != nil {
				fmt.Printf("Error unmarshalling JSON: %v\n", err)
				continue
			}

			// Check the compatibility of the metrics and add the results to the compatibility table
			metric1Available := metrics.Metric1 != ""
			metric2Available := metrics.Metric2 != ""
			metric3Available := metrics.Metric3 != ""

			compatibilityTable.WriteString(fmt.Sprintf(" | %v | %v | %v |\n", metric1Available, metric2Available, metric3Available))
		}
	}

	err = ioutil.WriteFile("COMPATIBILITY.md", []byte(compatibilityTable.String()), 0644)
	if err != nil {
		panic(err)
	}
}