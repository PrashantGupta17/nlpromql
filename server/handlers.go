package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/prashantgupta17/nlpromql/query_processing"
)

// handlePromQLQuery handles HTTP requests for PromQL queries.
func (s *PromQLServer) handlePromQLQuery(w http.ResponseWriter, r *http.Request) {
	// 1. Get User Query from Request
	userQuery := r.URL.Query().Get("query") // Assuming the query is passed as a URL parameter
	if userQuery == "" {
		http.Error(w, "Missing 'query' parameter", http.StatusBadRequest)
		return
	}

	// 2. Process User Query
	_, relevantMetrics, relevantLabels, relevantHistory, err := query_processing.ProcessUserQuery(
		s.openaiClient, userQuery, s.metricMap, s.labelMap,
		s.metricLabelMap, s.labelValueMap, s.nlpToMetricMap,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing query: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Generate PromQL Options
	promqlOptions, err := s.openaiClient.GetPromQLFromLLM(userQuery, relevantMetrics, relevantLabels, relevantHistory)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error generating PromQL: %v", err), http.StatusInternalServerError)
		return
	}

	// 4. Send JSON Response
	response := promqlOptions

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleReverseProxy forwards the request to another URL and returns the response.
func (s *PromQLServer) handleReverseProxy(w http.ResponseWriter, r *http.Request) {
	// The URL to which the request should be forwarded
	targetURL := "http://localhost:9090/api/v1/query" + "?" + r.URL.RawQuery
	revProxy(targetURL, w, r)
}

// handleReverseProxy forwards the request to another URL and returns the response.
func (s *PromQLServer) handleLabelReverseProxy(w http.ResponseWriter, r *http.Request) {
	// The URL to which the request should be forwarded
	targetURL := "http://localhost:9090/api/v1/label/__name__/values"
	revProxy(targetURL, w, r)
}

func revProxy(targetURL string, w http.ResponseWriter, r *http.Request) {
	url, err := url.Parse(targetURL)
	if err != nil {
		http.Error(w, "Error parsing target URL", http.StatusInternalServerError)
		return
	}

	proxyReq, err := http.NewRequest(r.Method, url.String(), r.Body)
	if err != nil {
		http.Error(w, "Error creating request to target", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header

	httpClient := &http.Client{}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}
