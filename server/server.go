package server

import (
	"fmt"
	"net/http"

	"github.com/prashantgupta17/nlpromql/info_structure"
	"github.com/prashantgupta17/nlpromql/llm"
)

type PromQLServer struct {
	llmClient      llm.LLMClient
	metricMap      info_structure.MetricMap
	labelMap       info_structure.LabelMap
	metricLabelMap info_structure.MetricLabelMap
	labelValueMap  info_structure.LabelValueMap
	nlpToMetricMap info_structure.NlpToMetricMap
}

func NewPromQLServer(llmClient llm.LLMClient, metricMap info_structure.MetricMap, labelMap info_structure.LabelMap,
	metricLabelMap info_structure.MetricLabelMap, labelValueMap info_structure.LabelValueMap, nlpToMetricMap info_structure.NlpToMetricMap) *PromQLServer {

	return &PromQLServer{
		llmClient:      llmClient,
		metricMap:      metricMap,
		labelMap:       labelMap,
		metricLabelMap: metricLabelMap,
		labelValueMap:  labelValueMap,
		nlpToMetricMap: nlpToMetricMap,
	}
}

func (s *PromQLServer) Start(port string) error {
	http.HandleFunc("/v1/promql", s.handlePromQLQuery)
	http.HandleFunc("/v1/query", s.handleReverseProxy)
	http.HandleFunc("/v1/label/__name__/values", s.handleLabelReverseProxy)

	fmt.Printf("Starting server on port %s...\n", port)
	return http.ListenAndServe(":"+port, nil)
}
