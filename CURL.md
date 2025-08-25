# LLM Test

## Curl examples

### Models list

```bash
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "OpenAI-Organization: org-3t9bdS1915T24Y6IlQCDt3Y3" \
  -H "OpenAI-Project: $PROJECT_ID"
```

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-3.5-turbo",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai"
    },
    {
      "id": "gpt-5-mini",
      "object": "model",
      "created": 1754425928,
      "owned_by": "system"
    },
    {
      "id": "gpt-4.1",
      "object": "model",
      "created": 1744316542,
      "owned_by": "system"
    }
  ]
}
```

### Chat

```bash
curl https://api.openai.com/v1/responses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4.1",
    "input": "Tell me a three sentence bedtime story about a unicorn."
  }'
```