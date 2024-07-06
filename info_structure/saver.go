package info_structure

import (
	"encoding/json"
	"fmt"
	"os"
)

// SaveInfoStructure saves all information structures to JSON files.
func (im *InfoStructureManager) SaveInfoStructure(metricMap MetricMap, labelMap LabelMap, metricLabelMap MetricLabelMap,
	labelValueMap LabelValueMap, nlpToMetricMap NlpToMetricMap) error {
	metricMapJSON := convertMetricMapToLists(metricMap)
	if err := saveMapToFile(im.PathToMetricMap, metricMapJSON); err != nil {
		return err
	}
	labelMapJSON := convertLabelMapToLists(labelMap)
	if err := saveMapToFile(im.PathToLabelMap, labelMapJSON); err != nil {
		return err
	}
	metricLabelMapJSON := convertMetricLabelMapToLists(metricLabelMap)
	if err := saveMapToFile(im.PathToMetricLabelMap, metricLabelMapJSON); err != nil {
		return err
	}
	labelValueMapJSON := convertLabelValueMapToLists(labelValueMap)
	if err := saveMapToFile(im.PathToLabelValueMap, labelValueMapJSON); err != nil {
		return err
	}
	if err := saveMapToFile(im.PathToNlpToMetricMap, nlpToMetricMap); err != nil {
		return err
	}
	return nil
}

// saveMapToFile saves a map to a JSON file.
func saveMapToFile(filePath string, data interface{}) error {
	fmt.Println("Saving:", filePath)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Indent for readability
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding JSON: %v", err)
	}

	return nil
}

// convertMetricMapToLists converts the metric names in a MetricMap from sets to lists for saving.
func convertMetricMapToLists(metricMap MetricMap) MetricJsonMap {
	result := MetricJsonMap{
		Map:      make(map[string][]string),
		AllNames: make([]string, 0, len(metricMap.AllNames)),
	}
	for synonym, metrics := range metricMap.Map {
		result.Map[synonym] = make([]string, 0, len(metrics))
		for s := range metrics {
			result.Map[synonym] = append(result.Map[synonym], s)
		}
	}
	for s := range metricMap.AllNames {
		result.AllNames = append(result.AllNames, s)
	}
	return result
}

// convertLabelMapToLists converts the label names in a LabelMap from sets to lists for saving.
func convertLabelMapToLists(labelMap LabelMap) LabelJsonMap {
	result := LabelJsonMap{
		Map:      make(map[string][]string),
		AllNames: make([]string, 0, len(labelMap.AllNames)),
	}
	for synonym, labels := range labelMap.Map {
		result.Map[synonym] = make([]string, 0, len(labels))
		for s := range labels {
			result.Map[synonym] = append(result.Map[synonym], s)
		}
	}
	for s := range labelMap.AllNames {
		result.AllNames = append(result.AllNames, s)
	}
	return result
}

// convertMetricLabelMapToLists converts the label values in a MetricLabelMap from sets to lists for saving.
func convertMetricLabelMapToLists(metricLabelMap MetricLabelMap) MapForJSON {
	result := make(MapForJSON)
	for metric, labelInfo := range metricLabelMap {
		result[metric] = make(MapForJSON)
		for label, values := range labelInfo.Labels {
			listValues := make([]string, 0, len(values.Values))
			for v := range values.Values {
				listValues = append(listValues, v)
			}
			result[metric].(MapForJSON)[label] = listValues
		}
	}
	return result
}

// convertLabelValueMapToLists converts the values in a LabelValueMap from sets to lists for saving.
func convertLabelValueMapToLists(labelValueMap LabelValueMap) MapForJSON {
	result := make(MapForJSON)
	for label, values := range labelValueMap {
		listValues := make([]string, 0, len(values.Values))
		for v := range values.Values {
			listValues = append(listValues, v)
		}
		result[label] = listValues
	}
	return result
}
