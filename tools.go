package main

type Tool struct {
	Name        string
	Description string
	Execute     func(input string) string
}

func Calculator(input string) string {
	switch input {
	case "25 * 4":
		return "100"
	case "2 + 2":
		return "4"
	default:
		return "I can only do simple math right now"
	}
}

var AvailableTools = []Tool{
	{
		Name:        "calculator",
		Description: "Does math calculations. Input should be a math expression like '2 + 2'",
		Execute:     Calculator,
	},
}
