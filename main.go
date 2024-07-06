package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/prashantgupta17/nlpromql/info_structure"
	"github.com/prashantgupta17/nlpromql/openai"
	"github.com/prashantgupta17/nlpromql/prometheus"
	"github.com/prashantgupta17/nlpromql/query_processing"
	"github.com/prashantgupta17/nlpromql/server"
)

func main() {
	mode := flag.String("mode", "server", "Mode of operation: 'server' or 'chat'")
	port := flag.String("port", "8080", "Port for the HTTP server (server mode only)")

	flag.Parse()

	openaiClient, err := openai.NewOpenAIClient()
	if err != nil {
		fmt.Println("Error initializing OpenAI client:", err)
		os.Exit(1)
	}

	// 3. Get Prometheus Credentials from Environment Variables
	promURL, promUser, promPassword, err := getPrometheusCredentials()
	if err != nil {
		fmt.Println("Error getting Prometheus credentials:", err)
		os.Exit(1)
	}

	// (You'll need to fill in the actual Prometheus URL, username, and password)
	promClient := prometheus.NewPrometheusConnect(promURL, promUser, promPassword)

	infoBuilder, err := info_structure.NewInfoBuilder(promClient, openaiClient, nil)
	if err != nil {
		fmt.Println("Error getting info builder:", err)
		os.Exit(1)
	}

	err = infoBuilder.BuildInformationStructure()
	if err != nil {
		fmt.Println("Error building information structure:", err)
		os.Exit(1)
	}
	fmt.Println("Information Structure Built Successfully:")
	fmt.Println("Metric Map:", len(infoBuilder.MetricMap.AllNames))
	fmt.Println("Metric Map:", len(infoBuilder.LabelMap.AllNames))
	fmt.Println("Metric Map:", len(*infoBuilder.MetricLabelMap))
	fmt.Println("Metric Map:", len(*infoBuilder.LabelValueMap))
	fmt.Println("Metric Map:", len(*infoBuilder.NlpToMetricMap))

	// 6. Main Loop for User Queries
	// Chat mode is disabled for now
	// runChatMode(openaiClient, metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap)

	switch *mode {
	case "server":
		promqlServer := server.NewPromQLServer(
			openaiClient,
			*infoBuilder.MetricMap,
			*infoBuilder.LabelMap,
			*infoBuilder.MetricLabelMap,
			*infoBuilder.LabelValueMap,
			*infoBuilder.NlpToMetricMap,
		)
		if err := promqlServer.Start(*port); err != nil {
			fmt.Println("Server error:", err)
			os.Exit(1)
		}
	case "chat":
		runChatMode(openaiClient,
			*infoBuilder.MetricMap,
			*infoBuilder.LabelMap,
			*infoBuilder.MetricLabelMap,
			*infoBuilder.LabelValueMap,
			*infoBuilder.NlpToMetricMap,
		)
	default:
		fmt.Println("Invalid mode. Use 'server' or 'chat'.")
		os.Exit(1)
	}
}

func runChatMode(openaiClient *openai.OpenAIClient, metricMap info_structure.MetricMap, labelMap info_structure.LabelMap,
	metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap,
	nlpToMetricMap info_structure.NlpToMetricMap) {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your query about Prometheus data (or type 'exit'): ")
		userQuery, _ := reader.ReadString('\n')
		userQuery = strings.TrimSpace(userQuery)

		if userQuery == "exit" {
			break
		}

		possibleMatches, relevantMetrics, relevantLabels, relevantHistory, err := query_processing.ProcessUserQuery(
			openaiClient, userQuery, metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap,
		)
		if err != nil {
			fmt.Println("Error processing user query:", err)
			continue
		}

		fmt.Println("Possible Matches:", possibleMatches)
		fmt.Println("Relevant Metrics:", relevantMetrics)
		fmt.Println("Relevant Labels:", relevantLabels)
		fmt.Println("Relevant History:", relevantHistory)

		promqlOptions, err := openaiClient.GetPromQLFromLLM(userQuery, relevantMetrics, relevantLabels, relevantHistory)
		if err != nil {
			fmt.Println("Error generating PromQL options:", err)
			continue
		}

		fmt.Println("Generated PromQL options:")
		for i, option := range promqlOptions {
			fmt.Printf("%d. %s\n", i+1, option)
		}
	}
}

// getPrometheusCredentials retrieves Prometheus credentials from environment variables.
func getPrometheusCredentials() (string, string, string, error) {
	promURL := os.Getenv("PROMETHEUS_URL")
	if promURL == "" {
		return "", "", "", fmt.Errorf("PROMETHEUS_URL environment variable not set")
	}

	promUser := os.Getenv("PROMETHEUS_USER")
	promPassword := os.Getenv("PROMETHEUS_PASSWORD")

	// Check if both username and password are provided (optional)
	if (promUser == "" && promPassword != "") || (promUser != "" && promPassword == "") {
		return "", "", "", fmt.Errorf("either both PROMETHEUS_USER and PROMETHEUS_PASSWORD should be set, or neither")
	}

	return promURL, promUser, promPassword, nil
}
