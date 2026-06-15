# GoFaux 2.0

GoFaux is a fully local mock API server for development and thesis experiments. It lets you define HTTP mocks manually or generate response JSON through a local AI provider such as Ollama, LM Studio, llama.cpp, or a deterministic no-model fallback.

## Current Features

1. Persistent JSON mock configuration.
2. Manual mock creation from the CLI.
3. Local AI/provider-assisted JSON response generation.
4. Request matching by method, path parameters, query parameters, headers, and priority.
5. Configurable status code, response headers, JSON body, and artificial delay.
6. Health and mock-inspection endpoints.
7. Local browser dashboard for creating, testing, inspecting, and deleting mocks.
8. Windows-friendly single executable packaging.
9. In-UI local model catalog, custom GGUF downloads, and progress tracking.
10. Runtime request analytics with captured headers, query strings, bodies, matched mocks, status codes, and timings.
11. OpenAPI/Swagger preview and import for contract-driven mock creation.
12. Constraint engine for patterns, ranges, enums, required fields, types, and schema-derived rules.
13. Local model evaluation dashboard for comparing JSON validity, semantic quality, fallback usage, and latency.
14. Project folder scanner that detects external API integrations, shows internal routes separately, and imports selected integrations as mocks.

## Project Structure

```text
internal/mock        core mock definitions, matching, storage, JSON helpers
internal/assistant   local AI providers, prompts, DTOs, template fallback
internal/httpserver  HTTP server, management endpoints, embedded web UI
internal/generator   shared AI generation workflow and intent parsing
internal/openapi     OpenAPI/Swagger parsing, schema extraction, import preview
internal/projectscan static project analysis for external API integration discovery
internal/cli         interactive terminal workflow
internal/config      environment configuration
internal/app         application wiring
```

## Run

```powershell
go run .
```

By default the app starts the local mock server and opens the browser dashboard:

```text
http://localhost:8080/_gofaux/ui/
```

Use the old terminal menu when needed:

```powershell
go run . --cli
```

Run without opening a browser:

```powershell
go run . --no-open
```

The app stores mocks in `gofaux.mocks.json` by default.

Change config path:

```powershell
$env:GOFAUX_CONFIG="examples/ecommerce.gofaux.json"
go run .
```

Change server address:

```powershell
$env:GOFAUX_ADDR=":8081"
go run .
```

## Local AI Providers

GoFaux is designed to stay free and local. The Go application does not need a paid cloud API. It can call a local model runtime on `localhost`, or use the built-in deterministic template provider when no model is installed.

The easiest path is to start the app and use:

```text
7 AI settings and model selection
```

This lets you choose a provider, discover local models when the runtime supports it, and save the choice to `.gofaux.settings.json`.

Built-in fallback:

```powershell
$env:GOFAUX_AI_PROVIDER="template"
```

Ollama:

```powershell
$env:GOFAUX_AI_PROVIDER="ollama"
$env:GOFAUX_AI_MODEL="llama3.2:3b"
$env:GOFAUX_AI_ENDPOINT="http://localhost:11434"
```

LM Studio, llama.cpp server, LocalAI, or another OpenAI-compatible local endpoint:

```powershell
$env:GOFAUX_AI_PROVIDER="openai"
$env:GOFAUX_AI_MODEL="local-model"
$env:GOFAUX_AI_ENDPOINT="http://localhost:1234/v1/chat/completions"
```

The mock server never calls AI during HTTP request handling. AI is used only while authoring mocks.

## Free Local Model Setup

Recommended free options:

1. **Ollama**: simple local model server. Pull a small model such as `llama3.2:3b` or another lightweight instruct model, then choose Ollama in GoFaux.
2. **LM Studio**: good for switching local GGUF models and exposing an OpenAI-compatible local server.
3. **llama.cpp server**: lightweight and scriptable, useful for advanced local experiments.
4. **Template provider**: no model required; useful as a baseline for thesis evaluation.

GoFaux uses local HTTP APIs so the Go app stays small, maintainable, and cross-platform. Running the full neural model directly inside Go is possible with native bindings, but it would make the thesis implementation heavier and harder to reproduce.

## Pulling Model Files Inside GoFaux

GoFaux also includes an in-app model downloader:

```text
8 Local model catalog/downloads
```

It can:

1. show a curated list of small free GGUF models,
2. download catalog models into `.gofaux/models`,
3. list downloaded catalog models,
4. download a custom GGUF URL.

This makes GoFaux responsible for model-file management instead of requiring the user to manually collect files.

Managed runner mode:

```text
7 AI settings and model selection
4 GoFaux managed runner
```

In this mode, GoFaux starts a local llama.cpp-compatible server against a downloaded GGUF model and uses it for response generation. The first run downloads the runner into `.gofaux/runners`.

This keeps the workflow free and local:

1. GoFaux downloads the model file.
2. GoFaux installs/starts the local runner.
3. GoFaux sends the DTO prompt to `localhost`.
4. The generated JSON is validated before saving the mock.

## Useful Endpoints

After starting the server:

```text
GET /_gofaux/ui/
GET /_gofaux/health
GET /_gofaux/mocks
GET /_gofaux/api/state
POST /_gofaux/api/generate
POST /_gofaux/api/mocks
DELETE /_gofaux/api/mocks/{id}
GET /_gofaux/api/traffic
DELETE /_gofaux/api/traffic
GET /_gofaux/api/models
POST /_gofaux/api/models/download
GET /_gofaux/api/models/downloads/{id}
POST /_gofaux/api/openapi/preview
POST /_gofaux/api/openapi/import
POST /_gofaux/api/project-scan/preview
POST /_gofaux/api/project-scan/import
GET /_gofaux/api/evaluations
POST /_gofaux/api/evaluations
GET /_gofaux/api/evaluations/{id}
```

Example:

```powershell
curl http://localhost:8080/_gofaux/health
```

## Build Windows EXE

```powershell
go build -o dist\GoFaux.exe .
```

Double-click `dist\GoFaux.exe` to start GoFaux and open the dashboard. The executable is self-contained for the Go application and embedded UI. Local runtime data such as downloaded GGUF models, runner files, settings, and mocks remain beside the app in `.gofaux`, `.gofaux.settings.json`, and `gofaux.mocks.json`.

## Request Analytics

GoFaux records recent mock traffic in memory while the application is running. The dashboard shows:

1. request method, path, query string, status, and duration,
2. matched mock ID/name/endpoint,
3. request headers and body preview,
4. top paths, status-code distribution, method distribution, and a request timeline.

The traffic log is intentionally local and runtime-scoped, so development request bodies are not sent anywhere or persisted unless you explicitly export them later.

## Project Integration Scanner

The dashboard includes a Project Scan workspace for pointing GoFaux at another backend or frontend project folder. The scanner performs local static analysis only. It looks for common outbound API client patterns such as Spring Feign, RestTemplate, WebClient, JavaScript `axios` and `fetch`, Go `net/http`, and Python `requests` or `httpx`. It also detects common incoming/internal route declarations such as Spring controllers, Express routes, and Go handlers.

Detected external integrations are checked by default. Internal routes are shown unchecked, so the developer can decide whether they should become mocks. Importing selected rows creates GoFaux mock definitions and can use either the current AI provider or the deterministic local template fallback.

## Thesis Notes

See `docs/ARCHITECTURE.md`, `docs/THESIS_ENGINEERING_NOTES.md`, and `docs/DEVELOPMENT_LOG.md` for implementation decisions, challenges, issue/fix notes, evaluation ideas, and thesis-ready wording.
