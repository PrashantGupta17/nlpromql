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
7. Output Format: You MUST return ONLY a valid JSON object with the following structure. Do NOT use markdown, do NOT call a function, do NOT include any text or explanation. Only output the JSON object as shown below.

{
  "possible_metric_names": ["metric1", "metric_synonym1", ...],
  "possible_label_names": ["label1", "label_synonym1", ...],
  "possible_label_values": ["value1", "value_synonym1", ...]
}
`
