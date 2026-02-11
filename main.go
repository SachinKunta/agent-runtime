package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// --- API Structures ---

type Message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []Message        `json:"messages"`
	Stream   bool             `json:"stream"`
	Tools    []ToolDefinition `json:"tools"`
}

type ChatResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

// --- Main ---

func main() {
	tools := GetToolDefinitions()

	messages := []Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant with tools: calculator (math), get_weather (weather by city), search (web search for facts). Use the right tool when needed. Do not guess answers.",
		},
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Agent ready. Type 'quit' to exit.")

	for {
		fmt.Print("\nYou: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "quit" || userInput == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if userInput == "" {
			continue
		}

		messages = append(messages, Message{
			Role:    "user",
			Content: userInput,
		})

		// Agent loop
		for {
			reqBody := ChatRequest{
				Model:    "llama3.1:8b",
				Messages: messages,
				Stream:   false,
				Tools:    tools,
			}

			jsonData, _ := json.Marshal(reqBody)
			resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				break
			}

			var chatResp ChatResponse
			json.NewDecoder(resp.Body).Decode(&chatResp)
			resp.Body.Close()

			messages = append(messages, chatResp.Message)
			toolCalls := chatResp.Message.ToolCalls

			if len(toolCalls) > 0 {
				for _, tc := range toolCalls {
					fmt.Printf("🔧 Using tool: %s\n", tc.Function.Name)

					result := ExecuteTool(tc.Function.Name, tc.Function.Arguments)
					fmt.Printf("📤 Result: %s\n", result)

					messages = append(messages, Message{
						Role:    "tool",
						Content: result,
					})
				}
				continue
			}

			fmt.Printf("\nAssistant: %s\n", chatResp.Message.Content)
			break
		}
	}
}
