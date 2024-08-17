package main

import (
	"fmt"
	"net/http"
	"strings"

	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

func main() {
	resp, err := http.DefaultClient.Get("http://black-box:5001/metrics")
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusOK {
		panic(resp.Status)
	}

	defer resp.Body.Close()

	// Create a parser
	parser := expfmt.TextParser{}
	// Parse the metrics
	parsedMetrics, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		fmt.Printf("Error parsing metrics: %v\n", err)
		return
	}

	labelName := "hostname"
	labelValue := "by-blackbox-machine"
	// re-label metrics
	for name, family := range parsedMetrics {
		// fmt.Printf("Metric Family: %s\n", name)
		for i := range family.Metric {
			// fmt.Printf("  Metric: %v\n", metric)
			if parsedMetrics[name].Metric[i].Label == nil {
				parsedMetrics[name].Metric[i].Label = []*io_prometheus_client.LabelPair{}
			}
			parsedMetrics[name].Metric[i].Label = append(parsedMetrics[name].Metric[i].Label,
				&io_prometheus_client.LabelPair{
					Name:  &labelName,
					Value: &labelValue,
				},
			)
		}
	}

	var output strings.Builder
	encoder := expfmt.NewEncoder(&output, expfmt.NewFormat(expfmt.TypeTextPlain))

	for _, family := range parsedMetrics {
		if err := encoder.Encode(family); err != nil {
			fmt.Printf("Error encoding metric family: %v\n", err)
			return
		}
	}

	// Display the encoded metrics
	// TODO: serve it via http
	fmt.Println(output.String())
}
