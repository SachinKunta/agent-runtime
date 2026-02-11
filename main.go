package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type Request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type StreamResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func buildPrompt(userQuery string, tools []Tool) string {
	prompt := `You are an AI assistant with access to tools.

Available tools:
`
	for _, tool := range tools {
		prompt += fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description)
	}

	prompt += `
When you need to use a tool, respond ONLY with this format (nothing else):
TOOL: <tool_name>
INPUT: <input>

Do NOT guess the output. Wait for the tool to respond.

User: ` + userQuery

	return prompt
}

func call(prompt string) string {
	req := Request{
		Model:  "llama3.1:8b",
		Prompt: prompt,
		Stream: true,
	}

	body, _ := json.Marshal(req)
	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "Error: " + err.Error()
	}
	defer resp.Body.Close()

	var fullResponse strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk StreamResponse
		json.Unmarshal(scanner.Bytes(), &chunk)
		fmt.Print(chunk.Response)
		fullResponse.WriteString(chunk.Response)
		if chunk.Done {
			fmt.Println()
			break
		}
	}
	return fullResponse.String()
}

// Parse tool call from model response
// Parse tool call from model response
func parseToolCall(response string) (toolName string, input string, found bool) {
	lines := strings.Split(response, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		// Check for "TOOL: x" format
		if strings.HasPrefix(upperLine, "TOOL:") {
			toolName = strings.TrimSpace(line[5:])
			toolName = strings.ToLower(toolName)
		} else if upperLine == "CALCULATOR" || upperLine == "TOOL" {
			// Model just said the tool name directly
			toolName = strings.ToLower(line)
		}

		// Check for "INPUT: x" format
		if strings.HasPrefix(upperLine, "INPUT:") {
			input = strings.TrimSpace(line[6:])
			found = toolName != ""
			return toolName, input, found
		}

		// Check next line for input if we found tool name
		if toolName != "" && i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if strings.HasPrefix(strings.ToUpper(nextLine), "INPUT:") {
				input = strings.TrimSpace(nextLine[6:])
				return toolName, input, true
			}
		}
	}

	return toolName, input, toolName != "" && input != ""
}

// Execute tool by name
func executeTool(toolName string, input string) string {
	for _, tool := range AvailableTools {
		if strings.ToLower(tool.Name) == toolName {
			return tool.Execute(input)
		}
	}
	return "Tool not found: " + toolName
}

func main() {
	userQuery := "What is 25 * 5?"

	fmt.Println("=== Step 1: Ask model ===")
	prompt := buildPrompt(userQuery, AvailableTools)
	response := call(prompt)

	// Check if model wants to use a tool
	toolName, input, found := parseToolCall(response)

	if found {
		fmt.Println("\n=== Step 2: Tool detected ===")
		fmt.Printf("Tool: %s\n", toolName)
		fmt.Printf("Input: %s\n", input)

		fmt.Println("\n=== Step 3: Execute tool ===")
		result := executeTool(toolName, input)
		fmt.Printf("Result: %s\n", result)

		fmt.Println("\n=== Step 4: Feed result back to model ===")
		followUp := prompt + "\n\nAssistant: " + response + "\n\nTool Result: " + result + "\n\nNow give the final answer to the user:"
		call(followUp)
	} else {
		fmt.Println("\n(No tool call detected)")
	}
}
