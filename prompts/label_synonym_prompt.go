package prompts

var LabelSynonymPrompt = `
Given a JSON array containing Prometheus label names, generate semantically related single-word synonyms or variations for each label that could be used in a monitoring and Prometheus context. 

Instructions:

1. **Single-Word Synonyms:** Generate only single-word synonyms. If a label name has multiple words separated by '_', split it into individual words and generate synonyms for each word.
2. **Nouns:** Consider both nouns and verbs when generating synonyms.
3. **Semantic Relevance:** Ensure that the generated synonyms are closely related in meaning to the original label name.
4. **Monitoring Context:**  The synonyms should be appropriate for use in dashboards, alerts, or other monitoring tools.
5. **Prometheus Conventions:** Follow standard Prometheus naming conventions and best practices.
6. **Variety:** Aim for a variety of synonyms to capture the full range of potential meanings and interpretations.
7. **Number of Synonyms:** Generate a minimum of 5 and a maximum of 10 synonyms for each label, depending on the complexity and potential for ambiguity.
8. **Output Consideration:** Output should always be in valid json format.

Metric Data:

%s

Output the results in JSON format:

{
  "original_label1": ["synonym1", "synonym2", ..., "synonym10"],
  "original_label2": ["synonym1", "synonym2", ..., "synonym5"],
  "original_label3": ["synonym1", "synonym2", ..., "synonym8"]
}
`
