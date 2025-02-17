# unifai-sdk-go

unifai-sdk-go is the Go SDK for Unifai, an AI native platform for dynamic tools and agent to agent communication.

## Installation

```bash
go get github.com/unifai-network/unifai-sdk-go
```

## Getting your Unifai API key

You can get your API key for free from [Unifai](https://app.unifai.network/).

There are two types of API keys:

- Agent API key: for using toolkits in your own agents.
- Toolkit API key: for creating toolkits that can be used by other agents.

## Using tools

To use tools in your agents, you need an **agent** API key. You can get an agent API key for free at [Unifai](https://app.unifai.network/).

```go
import (
    "github.com/unifai-network/unifai-sdk-go"
)

tools := unifai.NewTools(unifai.ToolsConfig{
    APIKey: "your-api-key",
})
```

Then you can pass the tools to any OpenAI compatible API. Popular options include:

- OpenAI's native API: For using OpenAI models directly
- [OpenRouter](https://openrouter.ai/docs): A service that gives you access to most LLMs through a single OpenAI compatible API

The tools will work with any API that follows the OpenAI function calling format. This gives you the flexibility to choose the best LLM for your needs while keeping your tools working consistently.

```go
messages := []openai.ChatCompletionMessageParamUnion{
    openai.UserMessage("Can you tell me what is trending on Google today?"),
}

params := openai.ChatCompletionNewParams{
    Model:    openai.F(openai.ChatModelGPT4o),
    Messages: openai.F(messages),
    Tools:    openai.F(tools.GetTools()),
}

completion, err := client.Chat.Completions.New(ctx, params)
```

If the response contains tool calls, you can pass them to the CallTools method to get the results. The output will be a list of messages containing the results of the tool calls that can be concatenated to the original messages and passed to the LLM again.

```go
results, err := tools.CallTools(ctx, completion.Choices[0].Message.ToolCalls)
for _, res := range results {
    messages = append(messages, res)
}
// messages can be passed to the LLM again now
```

Passing the tool calls results back to the LLM might get you more function calls, and you can keep calling 
the tools until you get a response that doesn't contain any tool calls. For example:

```go
messages := []openai.ChatCompletionMessageParamUnion{
    openai.UserMessage("What is happening in web3 today?"),
}

for {
    params := openai.ChatCompletionNewParams{
        Model:    openai.F(openai.ChatModelGPT4o),
        Messages: openai.F(messages),
        Tools:    openai.F(tools.GetTools()),
    }

    completion, err := client.Chat.Completions.New(ctx, params)
    if err != nil {
        // Handle error
    }

    message := completion.Choices[0].Message
    messages = append(messages, message)

    // If no tool calls, we're done
    if len(message.ToolCalls) == 0 {
        break
    }

    // Call the tools and get results
    results, err := tools.CallTools(ctx, message.ToolCalls)
    if err != nil {
        // Handle error
    }
    
    if len(results) == 0 {
        break
    }
    
    for _, res := range results {
        messages = append(messages, res)
    }
}
```

## Examples

You can find examples in the `examples` directory.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.
