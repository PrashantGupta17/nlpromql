package prompts

var ProcessQueryPrompt = `
Analyze the user query and provide possible matches for Prometheus metric names, label names, and label values.

User Query: %s

Your Task:

1. Identify potential metric names, label names, and label values relevant to the user query.
2. For each identified term, generate minimum 10 unique, semantically related synonyms or variations that could be used in a monitoring context. The generated result should only have single words without separators.
3. From the user query, ignore words that semantically mean like common PromQL keywords and functions like total, number, sum, count, avg, quantile, rate, irate, increase, topk, bottomk, time, all, any, etc. Ignore all stop words and punctuations.
4. If the query mentions a metric name, consider additional terms as potential label names.
5. If the query refers to a specific value along a label name, consider the value in potential possible label values. For e.g., "dev environment" or "prometheus server", then environment and server, are potential label names and dev and prometheus, are potential label values.
6. Some queries might only focus on labels and values, not needing a metric name. Usually these type of queries are where user asks to run an operation on a noun, for e.g. check everything for x, or give all for y. In these cases metric name is not needed.
7. Output should always be a valid json.

Output JSON format:
{
  "possible_metric_names": ["metric1", "metric_synonym1", "metric_synonym2", ...],
  "possible_label_names": ["label1", "label_synonym1", "label_synonym2", ...],
  "possible_label_values": ["value1", "value_synonym1", "value_synonym2", ...]
}

Examples:

**Query:** "Show the total number of requests for the payment service in the production environment."
**Output:**
{
  "possible_metric_names": ["request", "requests", "calls", "hits"],
  "possible_label_names": ["service", "app", "component", "environment", "env", "stage"],
  "possible_label_values": ["payment", "payments", "transactions", "production", "prod", "live"]
}

**Query:** "Find the 95th percentile latency for API calls in the staging cluster."
**Output:**
{
  "possible_metric_names": ["latency", "response_time", "duration", "percentile", "quantile"],
  "possible_label_names": ["api", "endpoint", "service", "cluster", "group", "job", "env", "environment"],
  "possible_label_values": ["staging", "test", "preprod"]
}

**Query:** "What is the average CPU usage for the database instances?"
**Output:**
{
  "possible_metric_names": ["cpu", "processor", "core", "usage", "utilization", "load"],
  "possible_label_names": ["instance", "host", "server", "database", "db", "job"],
  "possible_label_values": ["database", "mysql", "postgres"]
}

**Query:** "Memory consumption for pods in the 'kube-system' namespace."
**Output:**
{
  "possible_metric_names": ["memory", "RAM", "consumption", "usage", "used"],
  "possible_label_names": ["pod", "container", "namespace", "ns", "project"],
  "possible_label_values": ["kube-system"]
}

**Query:** "Node disk space." 
**Output:**
{
  "possible_metric_names": ["disk", "storage", "volume", "space", "capacity", "available"],
  "possible_label_names": ["node", "server", "instance", "device", "mountpoint"],
  "possible_label_values": [] 
}

**Query:** "List all nodes in the 'us-west-2' region."
**Output:**
{
  "possible_metric_names": [],  // Empty metric names
  "possible_label_names": ["node", "server", "instance", "region"],
  "possible_label_values": ["us-west-2"]
}

**Query:** "Show CPU usage for the 'web-server' job"
**Output:**
{
  "possible_metric_names": ["cpu", "usage"], 
  "possible_label_names": ["job"],
  "possible_label_values": ["web-server"]
}
`
