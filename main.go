package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func main() {
	req := Request{
		Model:  "llama3.1:8b",
		Prompt: "Explain what a goroutine is in 3 sentences.",
		Stream: true,
	}

	body, _ := json.Marshal(req)

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk StreamResponse
		json.Unmarshal(scanner.Bytes(), &chunk)
		fmt.Print(chunk.Response)
		if chunk.Done {
			fmt.Println()
			break
		}
	}
}
