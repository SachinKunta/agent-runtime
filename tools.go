package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Knetic/govaluate"
)

// --- Tool Definitions ---

type ToolDefinition struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Parameters  Parameters `json:"parameters"`
}

type Parameters struct {
	Type       string           `json:"type"`
	Properties map[string]Param `json:"properties"`
	Required   []string         `json:"required"`
}

type Param struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

func GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: Function{
				Name:        "calculator",
				Description: "Useful for performing math. Input is a string expression.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Param{
						"expression": {Type: "string", Description: "The math to solve, e.g. '25 * 5'"},
					},
					Required: []string{"expression"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "get_weather",
				Description: "Gets current weather for a city. Use this when user asks about weather.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Param{
						"city": {Type: "string", Description: "City name, e.g. 'New York'"},
					},
					Required: []string{"city"},
				},
			},
		},
		{
			Type: "function",
			Function: Function{
				Name:        "search",
				Description: "Search the web for information. Use for factual questions about people, places, concepts.",
				Parameters: Parameters{
					Type: "object",
					Properties: map[string]Param{
						"query": {Type: "string", Description: "The search query"},
					},
					Required: []string{"query"},
				},
			},
		},
	}
}

// --- Tool Implementations ---

func ExecuteTool(name string, args map[string]any) string {
	switch name {
	case "calculator":
		expr := args["expression"].(string)
		return calculate(expr)
	case "get_weather":
		city := args["city"].(string)
		return getWeather(city)
	case "search":
		query := args["query"].(string)
		return search(query)
	default:
		return "Unknown tool: " + name
	}
}

func calculate(expression string) string {
	expr, err := govaluate.NewEvaluableExpression(expression)
	if err != nil {
		return "Error parsing expression: " + err.Error()
	}
	result, err := expr.Evaluate(nil)
	if err != nil {
		return "Error calculating: " + err.Error()
	}
	return fmt.Sprintf("%v", result)
}

func getWeather(city string) string {
	geoURL := fmt.Sprintf("https://geocoding-api.open-meteo.com/v1/search?name=%s&count=1", url.QueryEscape(city))
	geoResp, err := http.Get(geoURL)
	if err != nil {
		return "Error: " + err.Error()
	}
	defer geoResp.Body.Close()

	var geoData struct {
		Results []struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
			Name      string  `json:"name"`
		} `json:"results"`
	}
	json.NewDecoder(geoResp.Body).Decode(&geoData)

	if len(geoData.Results) == 0 {
		return "City not found: " + city
	}

	lat := geoData.Results[0].Latitude
	lon := geoData.Results[0].Longitude
	name := geoData.Results[0].Name

	weatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current_weather=true", lat, lon)
	weatherResp, err := http.Get(weatherURL)
	if err != nil {
		return "Error: " + err.Error()
	}
	defer weatherResp.Body.Close()

	var weatherData struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
			WindSpeed   float64 `json:"windspeed"`
		} `json:"current_weather"`
	}
	json.NewDecoder(weatherResp.Body).Decode(&weatherData)

	return fmt.Sprintf("%s: %.1f°C, wind %.1f km/h", name, weatherData.CurrentWeather.Temperature, weatherData.CurrentWeather.WindSpeed)
}

func search(query string) string {
	searchURL := fmt.Sprintf("https://en.wikipedia.org/w/api.php?action=opensearch&search=%s&limit=1&format=json", url.QueryEscape(query))

	client := &http.Client{}

	req, _ := http.NewRequest("GET", searchURL, nil)
	req.Header.Set("User-Agent", "AgentRuntime/1.0 (learning project)")

	resp, err := client.Do(req)
	if err != nil {
		return "Error: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var results []json.RawMessage
	json.Unmarshal(body, &results)

	if len(results) < 4 {
		return "No results found for: " + query
	}

	var titles []string
	json.Unmarshal(results[1], &titles)

	var descriptions []string
	json.Unmarshal(results[2], &descriptions)

	if len(titles) == 0 {
		return "No results found for: " + query
	}

	if len(descriptions) > 0 && descriptions[0] != "" {
		return descriptions[0]
	}

	// Fetch full summary
	title := titles[0]
	summaryURL := fmt.Sprintf("https://en.wikipedia.org/api/rest_v1/page/summary/%s", strings.ReplaceAll(title, " ", "_"))

	summaryReq, _ := http.NewRequest("GET", summaryURL, nil)
	summaryReq.Header.Set("User-Agent", "AgentRuntime/1.0 (learning project)")

	summaryResp, err := client.Do(summaryReq)
	if err != nil {
		return "Found: " + title + " (couldn't fetch details)"
	}
	defer summaryResp.Body.Close()

	summaryBody, _ := io.ReadAll(summaryResp.Body)

	var summary struct {
		Extract string `json:"extract"`
	}
	json.Unmarshal(summaryBody, &summary)

	if summary.Extract != "" {
		if len(summary.Extract) > 500 {
			return summary.Extract[:500] + "..."
		}
		return summary.Extract
	}

	return "Found: " + title
}
