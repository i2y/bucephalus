// Package main demonstrates tool calling with Bucephalus.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/i2y/bucephalus/llm"
	_ "github.com/i2y/bucephalus/openai" // Register OpenAI provider
)

// WeatherInput defines the input parameters for the weather tool.
type WeatherInput struct {
	City    string `json:"city" jsonschema:"required,description=The city name"`
	Country string `json:"country,omitempty" jsonschema:"description=The country code (e.g. US, JP)"`
}

// WeatherOutput defines the output of the weather tool.
type WeatherOutput struct {
	Temperature float64 `json:"temperature"`
	Unit        string  `json:"unit"`
	Conditions  string  `json:"conditions"`
	Humidity    int     `json:"humidity"`
}

// CalculatorInput defines the input for the calculator tool.
type CalculatorInput struct {
	Operation string  `json:"operation" jsonschema:"required,description=The operation: add, subtract, multiply, divide"`
	A         float64 `json:"a" jsonschema:"required,description=First number"`
	B         float64 `json:"b" jsonschema:"required,description=Second number"`
}

// CalculatorOutput defines the output of the calculator tool.
type CalculatorOutput struct {
	Result float64 `json:"result"`
}

func main() {
	ctx := context.Background()

	// Create type-safe tools
	weatherTool := llm.MustNewTool("get_weather", "Get the current weather for a city",
		func(ctx context.Context, in WeatherInput) (WeatherOutput, error) {
			// In a real app, this would call a weather API
			fmt.Printf("[Tool called] get_weather: city=%s, country=%s\n", in.City, in.Country)
			return WeatherOutput{
				Temperature: 22.5,
				Unit:        "celsius",
				Conditions:  "Partly cloudy",
				Humidity:    65,
			}, nil
		},
	)

	calculatorTool := llm.MustNewTool("calculator", "Perform basic math operations",
		func(ctx context.Context, in CalculatorInput) (CalculatorOutput, error) {
			fmt.Printf("[Tool called] calculator: %s(%g, %g)\n", in.Operation, in.A, in.B)
			var result float64
			switch in.Operation {
			case "add":
				result = in.A + in.B
			case "subtract":
				result = in.A - in.B
			case "multiply":
				result = in.A * in.B
			case "divide":
				if in.B == 0 {
					return CalculatorOutput{}, fmt.Errorf("division by zero")
				}
				result = in.A / in.B
			default:
				return CalculatorOutput{}, fmt.Errorf("unknown operation: %s", in.Operation)
			}
			return CalculatorOutput{Result: result}, nil
		},
	)

	// Create a tool registry
	registry := llm.NewToolRegistry()
	registry.Register(weatherTool, calculatorTool)

	// Make a call with tools
	fmt.Println("Asking about weather...")
	resp, err := llm.Call(ctx, "What's the weather like in Tokyo, Japan?",
		llm.WithProvider("openai"),
		llm.WithModel("gpt-4o-mini"),
		llm.WithTools(weatherTool, calculatorTool),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if the model wants to call tools
	if resp.HasToolCalls() {
		fmt.Println("\nModel requested tool calls:")
		for _, tc := range resp.ToolCalls() {
			fmt.Printf("  - %s: %s\n", tc.Name, tc.Arguments)
		}

		// Execute the tool calls
		toolMessages, err := llm.ExecuteToolCalls(ctx, resp.ToolCalls(), registry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing tools: %v\n", err)
			os.Exit(1)
		}

		// Continue the conversation with tool results
		fmt.Println("\nContinuing conversation with tool results...")
		messages := []llm.Message{
			llm.UserMessage("What's the weather like in Tokyo, Japan?"),
			llm.AssistantMessageWithToolCalls("", resp.ToolCalls()),
		}
		messages = append(messages, toolMessages...)

		resp2, err := llm.CallMessages(ctx, messages,
			llm.WithProvider("openai"),
			llm.WithModel("gpt-4o-mini"),
			llm.WithTools(weatherTool, calculatorTool),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nFinal response:")
		fmt.Println(resp2.Text())
	} else {
		fmt.Println("\nResponse (no tool calls):")
		fmt.Println(resp.Text())
	}

	// Demonstrate TypedCall - direct tool invocation without JSON
	fmt.Println("\n--- Direct TypedCall demo ---")
	out, err := weatherTool.TypedCall(ctx, WeatherInput{City: "Paris", Country: "FR"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	result, _ := json.MarshalIndent(out, "", "  ")
	fmt.Printf("Direct call result:\n%s\n", result)
}
