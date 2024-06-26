package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/prashantgupta17/nlpromql/config"
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

	// 1. Load Configuration and Prompts
	_, _, _, _, err := config.LoadPrompts() // Load prompts to ensure they are available
	if err != nil {
		fmt.Println("Error loading prompts:", err)
		os.Exit(1)
	}

	// 2. Initialize OpenAI Client
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

	// 3. Initialize Prometheus Client
	// (You'll need to fill in the actual Prometheus URL, username, and password)
	promClient := prometheus.NewPrometheusConnect(promURL, promUser, promPassword)

	// 4. Build Information Structure
	metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap, err := info_structure.BuildInformationStructure(promClient, openaiClient)
	if err != nil {
		fmt.Println("Error building information structure:", err)
		os.Exit(1)
	}

	// 5. Print the Result (Optional)
	fmt.Println("Information Structure Built Successfully:")
	fmt.Println("Metric Map:", len(metricMap.AllNames))
	fmt.Println("Metric Map:", len(labelMap.AllNames))
	fmt.Println("Metric Map:", len(metricLabelMap))
	fmt.Println("Metric Map:", len(labelValueMap))
	fmt.Println("Metric Map:", len(nlpToMetricMap))

	// 6. Main Loop for User Queries
	// Chat mode is disabled for now
	// runChatMode(openaiClient, metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap)

	switch *mode {
	case "server":
		promqlServer := server.NewPromQLServer(
			openaiClient,
			metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap,
		)
		if err := promqlServer.Start(*port); err != nil {
			fmt.Println("Server error:", err)
			os.Exit(1)
		}
	case "chat":
		runChatMode(openaiClient, metricMap, labelMap, metricLabelMap, labelValueMap, nlpToMetricMap)
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
