# axon-code

`axon-code` is a Go library that implements a coding agent. It drives an LLM conversation loop to implement plan steps against a project directory, using file-system and shell tools sandboxed to that directory.

## Import

```go
import code "github.com/benaskins/axon-code"
```

## Usage

```go
package main

import (
    "fmt"
    "os"

    code "github.com/benaskins/axon-code"
    "github.com/benaskins/axon-code/plan"
    talk "github.com/benaskins/axon-talk"
)

func main() {
    client := talk.NewAnthropicClient(os.Getenv("ANTHROPIC_API_KEY"))

    coder := code.New(client,
        code.WithMaxIterations(30),
        code.WithVerbose(os.Stdout),
    )

    step := plan.Step{
        Title:       "Add greeting handler",
        Description: "Add a /hello endpoint that returns 'Hello, world!' with status 200.",
    }

    summary, err := coder.Implement("/path/to/project", step, "")
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Println(summary)
}
```

## Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithMaxIterations(n)` | 50 | Maximum LLM turns per `Implement` call |
| `WithTimeout(d)` | 15m | Wall-clock timeout per `Implement` call |
| `WithSystemPromptPrefix(s)` | `""` | Prepended to the system prompt |
| `WithVerbose(w)` | nil | Writer for tool-use trace output |

## Prerequisites

- Go 1.24+
- [just](https://github.com/casey/just)

## Development

```bash
just build   # go build ./...
just test    # go test ./...
just lint    # go vet ./...
```
