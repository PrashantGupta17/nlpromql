package prompts

var MetricSynonymPrompt = `
Given a JSON object containing Prometheus metric names and their descriptions, generate semantically related single-word synonyms or variations for each metric that could be used in a monitoring and Prometheus context. 

Instructions:

1. **Single-Word Synonyms:** Generate only single-word synonyms. If a metric name has multiple words separated by '_', split it into individual words and generate synonyms for each word.
2. **Nouns:** Consider both nouns and verbs when generating synonyms.
3. **Semantic Relevance:** Ensure that the generated synonyms are closely related in meaning to the original metric name and its description.
4. **Monitoring Context:**  The synonyms should be appropriate for use in dashboards, alerts, or other monitoring tools.
5. **Prometheus Conventions:** Follow standard Prometheus naming conventions and best practices.
6. **Variety:** Aim for a variety of synonyms to capture the full range of potential meanings and interpretations.
7. **Number of Synonyms:** Generate a minimum of 5 and a maximum of 10 synonyms for each metric, depending on the complexity and potential for ambiguity. Do not repeat synonyms.
8. **Description Consideration:** Prioritize the metric description (if provided) for semantic relevance. If the description is empty, ignore it.
9. **Output Consideration:** Output should always be in valid json format.

Metric Data:

%s

Output the results in JSON format:

{{
  "original_metric1": ["synonym1", "synonym2", ..., "synonym10"],
  "original_metric2": ["synonym1", "synonym2", ..., "synonym5"],
  "original_metric3": ["synonym1", "synonym2", ..., "synonym8"]
}}
`
