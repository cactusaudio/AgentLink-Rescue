package brain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

func ExtractJSONObject(text string) (string, error) {
	text = sanitizeModelOutput(text)
	start := strings.Index(text, "{")
	if start < 0 {
		return "", fmt.Errorf("no JSON object found")
	}
	candidate, end, err := extractObjectAt(text, start)
	if err != nil {
		return "", err
	}
	if next := strings.Index(text[end:], "{"); next >= 0 {
		if _, err := ExtractJSONObject(text[end+next:]); err == nil {
			return "", fmt.Errorf("multiple top-level JSON objects found")
		}
	}
	return candidate, nil
}

func ExtractPlannerJSONObject(text string) (string, error) {
	text = sanitizeModelOutput(text)
	var plannerObjects []string
	var lastValid string
	for i := 0; i < len(text); i++ {
		if text[i] != '{' {
			continue
		}
		candidate, end, err := extractObjectAt(text, i)
		if err != nil {
			continue
		}
		lastValid = candidate
		if looksLikePlannerDecision(candidate) {
			plannerObjects = append(plannerObjects, candidate)
		}
		i = end - 1
	}
	if len(plannerObjects) > 1 {
		return "", fmt.Errorf("multiple planner JSON objects found")
	}
	if len(plannerObjects) == 1 {
		return plannerObjects[0], nil
	}
	if lastValid != "" {
		return lastValid, nil
	}
	return "", fmt.Errorf("no JSON object found")
}

func extractObjectAt(text string, start int) (string, int, error) {
	depth := 0
	inString := false
	escaped := false
	end := -1
	for i := start; i < len(text); i++ {
		c := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
				i = len(text)
			}
			if depth < 0 {
				return "", 0, fmt.Errorf("malformed JSON braces")
			}
		}
	}
	if end < 0 || depth != 0 {
		return "", 0, fmt.Errorf("unterminated JSON object")
	}
	candidate := text[start:end]
	var tmp any
	if err := json.Unmarshal([]byte(candidate), &tmp); err != nil {
		return "", 0, err
	}
	return candidate, end, nil
}

func looksLikePlannerDecision(text string) bool {
	return strings.Contains(text, `"schemaVersion"`) && strings.Contains(text, `"intent"`)
}

func sanitizeModelOutput(text string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	text = re.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "```json", "")
	text = strings.ReplaceAll(text, "```JSON", "")
	text = strings.ReplaceAll(text, "```", "")
	return text
}
