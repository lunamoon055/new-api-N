# Creation Center Model Catalog API

The creation center uses a dedicated read-only model catalog endpoint:

```http
GET /api/creation/models
GET /api/creation/models?mode=chat
GET /api/creation/models?mode=image
GET /api/creation/models?mode=video
```

The endpoint supports anonymous browsing and optional user sessions. Authenticated
users receive models filtered by their usable groups. Anonymous visitors receive
models from the publicly usable groups. Sending generation requests remains an
authenticated operation handled by the existing relay endpoints.

The response intentionally excludes pricing ratios, billing expressions, channel
IDs, and group configuration:

```json
{
  "success": true,
  "message": "",
  "data": {
    "modes": [
      {
        "mode": "chat",
        "models": [
          {
            "id": "gpt-5.4-mini",
            "description": "Lightweight model for everyday tasks.",
            "icon": "OpenAI",
            "tags": ["recommended", "fast"],
            "vendor_id": 1,
            "supported_endpoint_types": ["openai"]
          }
        ]
      }
    ],
    "vendors": [
      {
        "id": 1,
        "name": "OpenAI",
        "icon": "OpenAI"
      }
    ]
  }
}
```

Models are classified from their supported endpoint types:

- `chat`: OpenAI-compatible, Responses, Anthropic, or Gemini endpoints
- `image`: image generation endpoints
- `video`: OpenAI-compatible video endpoints

Image and video endpoint types take precedence over chat-compatible fallbacks so
that dedicated generation models appear in the correct creation mode.
