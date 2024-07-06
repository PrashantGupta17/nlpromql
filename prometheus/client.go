package prometheus

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PrometheusConnect provides methods to interact with the Prometheus API.
type PrometheusConnect struct {
	url    string
	user   string
	pass   string
	client *http.Client
}

// NewPrometheusConnect creates a new PrometheusConnect client.
func NewPrometheusConnect(url, username, password string) *PrometheusConnect {
	return &PrometheusConnect{
		url:    url,
		user:   username,
		pass:   password,
		client: &http.Client{Timeout: 120 * time.Second}, // Adjust timeout as needed
	}
}

// all_metrics fetches all metric names from Prometheus.
func (p *PrometheusConnect) AllMetrics() ([]string, error) {
	endpoint := p.url + "/api/v1/label/__name__/values"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching all metrics: %v", err)
	}
	req.SetBasicAuth(p.user, p.pass)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching all metrics: %v", err)
	}
	defer resp.Body.Close()

	var result AllMetricsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding all metrics response: %v", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus API error: %s", result.Status)
	}

	return result.Data, nil
}

// all_metrics fetches all metric names from Prometheus.
func (p *PrometheusConnect) AllLabels() ([]string, error) {
	endpoint := p.url + "/api/v1/labels"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching all labels: %v", err)
	}
	req.SetBasicAuth(p.user, p.pass)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching all labels: %v", err)
	}
	defer resp.Body.Close()

	var result AllLabelsResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding all labels response: %v", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus API error: %s", result.Status)
	}

	return result.Data, nil
}

// custom_query performs a custom PromQL query against Prometheus.
func (p *PrometheusConnect) CustomQuery(query string) ([]Metric, error) {
	endpoint := p.url + "/api/v1/query?query=" + url.QueryEscape(query) + "&time=" + strconv.FormatInt(time.Now().Unix(), 10)
	fmt.Println("Querying:", endpoint)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating query: %v", err)
	}
	req.SetBasicAuth(p.user, p.pass)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string   `json:"resultType"`
			Result     []Metric `json:"result"`
		} `json:"data"`
	}
	// fmt.Println(resp.Body)
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding query response: %v", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus API error: %s", result.Status)
	}

	return result.Data.Result, nil
}

// AllMetadata fetches metadata for all metrics from Prometheus.
func (p *PrometheusConnect) AllMetadata() (map[string]string, error) {
	endpoint := p.url + "/api/v1/metadata"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching metadata: %v", err)
	}
	req.SetBasicAuth(p.user, p.pass)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching metadata: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Status string `json:"status"`
		Data   map[string][]struct {
			Type string `json:"type"`
			Help string `json:"help"`
			Unit string `json:"unit"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding metadata response: %v", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus API error: %s", result.Status)
	}

	metadata := make(map[string]string)
	for metricName, infos := range result.Data {
		if len(infos) > 0 {
			// Assuming the first entry contains the relevant description for simplicity.
			metadata[metricName] = infos[0].Help
		}
	}

	return metadata, nil
}
