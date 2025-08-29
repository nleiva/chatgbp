# ChatGBT - Educational ChatGPT Clone
[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/nleiva/chatgbt?quickstart=1)

A simple ChatGPT clone built for educational purposes, supporting CLI, web, and direct query modes.

## Prerequisites

- Go 1.25 or later (json/v2)
- LLM Provider API key

## Installation

1. Clone the repository:
```bash
git clone https://github.com/nleiva/chatgbp.git
cd chatgbt
```

2. Build the application:
```bash
make build
```

## Usage

### Environment Variables

- `API_KEY` (required): Your LLM Provider API key
- `MODEL` (optional): Model to use (default: gpt-3.5-turbo)
- `PORT` (optional): Port for web server (default: 3000)
- `TOKEN_BUDGET` (optional): Session token budget (default: 10000)
- `COST_BUDGET` (optional): Session cost budget in USD (default: $0.02)

### CLI Mode

Interactive terminal interface:

## Quick Start

```bash
# Set your API key
export API_KEY="your-key-here"

# Optional: Set LLM provider (defaults to openai)
export LLM_PROVIDER="openai"  # or "anthropic", "bedrock"

make run-cli
```

#### CLI Commands

- Type your message and press Enter twice (empty line) to send
- `exit` - Quit the application
- `/reset` - Reset the conversation
- `/system` - Update the system prompt
- `/budget` - Check token and cost budget status
- `/stats` - Show session statistics
- `/prune` - Manually prune conversation context

### Web Mode

Start the web server:

```bash
export OPENAI_API_KEY="your-api-key-here"
make run-web
```

The web interface will be available at `http://localhost:3000`

### Direct Query Mode

For quick, one-off queries:

```bash
export OPENAI_API_KEY="your-api-key-here"
./chatgbt "explain Go channels"
./chatgbt "debug this code: [paste your code]"
```

## Technologies Used

- **Backend**: Go with modular architecture
- **CLI**: Standard Go libraries for terminal interaction
- **Web Frontend**: 
  - [Fiber](https://github.com/gofiber/fiber) - Fast HTTP framework
  - [Templ](https://github.com/a-h/templ) - Type-safe HTML templates
  - [HTMX](https://htmx.org/) - Dynamic web interactions
- **API**: OpenAI Chat Completions API
