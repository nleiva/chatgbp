# ChatGBT - Educational ChatGPT Clone
[![Open in GitHub Codespaces](https://github.com/codespaces/badge.svg)](https://codespaces.new/nleiva/chatgbt?quickstart=1)

A simple ChatGPT clone built for educational purposes, supporting CLI, web, and direct query modes.

## Prerequisites

- Go 1.24 or later
- OpenAI API key

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd chatGBT
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o chatgbt .
# Or using Make
make build
```

## Usage

### Environment Variables

- `OPENAI_API_KEY` (required): Your OpenAI API key
- `MODEL` (optional): Model to use (default: gpt-3.5-turbo)
- `PORT` (optional): Port for web server (default: 3000)
- `TOKEN_BUDGET` (optional): Session token budget (default: 10000)
- `COST_BUDGET` (optional): Session cost budget in USD (default: $0.02)

### CLI Mode

Interactive terminal interface:

```bash
export OPENAI_API_KEY="your-api-key-here"
./chatgbt cli
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
./chatgbt web
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

## License

This project is for educational purposes. Please respect OpenAI's usage policies when using their API.
