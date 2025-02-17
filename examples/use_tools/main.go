package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/openai/openai-go"
	"github.com/unifai-network/unifai-sdk-go"
)

// systemPrompt defines the system instructions.
var systemPrompt = `
You are a personal assistant capable of doing many things with your tools. When you are given a task you cannot finish right now (like something you don't know, or requires you to take some action), try find appropriate tools to do it.

When searching for tools, try to think what tools might be useful and use relevant generic keywords rather than putting all the details/numbers into the query, because you are finding the tool, not solving the problem itself. If you failed to find the appropriate tools, you can try changing query and search again.

Sometimes there are multiple tools that can be used to finish a task. If the result from one tool is not good enough, you can try to use other tools and compare or combine results. Specifically, when you are trying to get comprehensive data, information, or analysis, using results from multiple tools or sources are beneficial.

When you search for a tool, before actually call the tool, tell user what tools you got from search tool results, which one you picked.
`

// run performs the main conversation loop with the OpenAI API and tool calls.
func run(msg string) error {
	ctx := context.Background()

	// Create a new Tools instance with your agent API key.
	tools := unifai.NewTools(unifai.ToolsConfig{
		APIKey: "YOUR_AGENT_API_KEY",
	})

	// Create a new OpenAI client.
	client := openai.NewClient()

	// Initialize the conversation with a system message and the user's message.
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(msg),
	}

	for {
		// Prepare the completion request, including the tool definitions.
		params := openai.ChatCompletionNewParams{
			Model:    openai.F(openai.ChatModelGPT4o),
			Messages: openai.F(messages),
			Tools:    openai.F(tools.GetTools()),
		}

		// Call the OpenAI chat completions endpoint.
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to get completion: %w", err)
		}

		// Ensure there is at least one choice.
		if len(completion.Choices) == 0 {
			return fmt.Errorf("no choices returned")
		}

		// Use the first returned message.
		message := completion.Choices[0].Message

		// If the message contains textual content, print it.
		if len(message.Content) > 0 {
			fmt.Println(message.Content)
		}

		// Append the returned message to our conversation.
		messages = append(messages, message)

		// If no tool calls are returned, we're done.
		if len(message.ToolCalls) == 0 {
			break
		}

		// Print the tool calls.
		fmt.Printf("Calling tools:")
		for _, tc := range message.ToolCalls {
			fmt.Printf(" %s(%s)", tc.Function.Name, tc.Function.Arguments)
		}
		fmt.Println()
		// Call the tools concurrently.
		results, err := tools.CallTools(ctx, message.ToolCalls)
		if err != nil {
			return fmt.Errorf("failed to call tools: %w", err)
		}

		// If no results, break the loop.
		if len(results) == 0 {
			break
		}

		for _, res := range results {
			messages = append(messages, res)
		}
	}

	return nil
}

func main() {
	// Ensure a message is provided via command-line arguments.
	if len(os.Args) < 2 {
		fmt.Println("Please provide a message")
		os.Exit(1)
	}
	msg := strings.Join(os.Args[1:], " ")
	// Run the conversation.
	if err := run(msg); err != nil {
		log.Fatal(err)
	}
}
