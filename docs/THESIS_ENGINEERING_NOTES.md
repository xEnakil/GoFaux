# GoFaux 2.0 Engineering Notes for the Thesis

This file records implementation decisions, challenges, and thesis-ready observations. It is written so another conversation can read it later and continue the thesis without rediscovering the engineering context.

For chronological issue/fix notes, see `docs/DEVELOPMENT_LOG.md`.

## Current Development State

The initial project was a small command-line MVP:

1. In-memory list of mocked endpoints.
2. Manual endpoint creation.
3. Basic JSON syntax validation.
4. A global HTTP server on port 8080.
5. Exact matching only by HTTP method and path.

The current direction is **GoFaux 2.0**, a local-first mock API server designed for Master thesis evaluation. The new architecture separates storage, request matching, serving, and AI-assisted authoring.

## Current Module Structure

The project was refactored into a production-style Go layout:

1. `internal/mock` contains the core artifact logic: mock definitions, matching, JSON validation, and persistent storage.
2. `internal/assistant` contains local AI provider interfaces, prompt DTOs, Ollama integration, OpenAI-compatible local endpoint integration, and deterministic fallback generation.
3. `internal/httpserver` contains runtime HTTP serving and management endpoints.
4. `internal/cli` contains the terminal workflow.
5. `internal/config` contains environment-based configuration.
6. `internal/app` wires dependencies together.
7. `main.go` is only the process entry point.

Thesis sentence:

> The final implementation follows a modular Go structure in which the core mock domain is isolated from user interface, HTTP serving, configuration, and assistant-provider concerns. This makes the artifact easier to test, explain, and extend.

## Implemented Architectural Decisions

### 1. Persistent local configuration

Mock definitions are stored in a JSON file, defaulting to `gofaux.mocks.json`. The path can be changed with:

```powershell
$env:GOFAUX_CONFIG="examples/ecommerce.gofaux.json"
```

This supports thesis requirements around reproducibility: the same mock set can be loaded, inspected, shared, and benchmarked.

### 2. Request matching model

Each mock can match:

1. HTTP method.
2. Path pattern, including parameters such as `/users/{id}`.
3. Required query parameters.
4. Required request headers.
5. Priority, where higher priority wins.

This is intentionally more advanced than the MVP but still simple enough to explain clearly in the thesis. It is suitable for local development scenarios where several mocks may share a path but differ by query/header conditions.

### 3. Response model

Each mock can define:

1. HTTP status code.
2. Response headers.
3. JSON body.
4. Artificial delay in milliseconds.
5. Description, name, tags, priority, and runtime request counter.

The runtime request counter is kept in memory during serving. This avoids writing to disk for every request, which would distort performance benchmarks.

### 4. AI generation is outside the request path

The local AI/provider is used only during mock authoring. The HTTP server never calls an AI model while answering client requests.

This design matters because it keeps runtime latency predictable. It also makes the benchmark fair: performance results measure request matching and serving, not model inference.

### 5. Provider abstraction for local AI experiments

The assistant layer defines a provider interface:

```go
type Provider interface {
    Name() string
    GenerateMock(ctx context.Context, req GenerateRequest) (GenerateResult, error)
    RepairJSON(ctx context.Context, invalid string, validationError string, req GenerateRequest) (GenerateResult, error)
}
```

The app can use:

1. `template` provider: deterministic local fallback, no model required.
2. `ollama` provider: local Ollama HTTP API.
3. `openai`, `lmstudio`, `llamacpp`, or `localai`: OpenAI-compatible local server API.

Environment variables:

```powershell
$env:GOFAUX_AI_PROVIDER="ollama"
$env:GOFAUX_AI_MODEL="llama3.2:3b"
$env:GOFAUX_AI_ENDPOINT="http://localhost:11434"
```

For LM Studio or llama.cpp server:

```powershell
$env:GOFAUX_AI_PROVIDER="openai"
$env:GOFAUX_AI_MODEL="local-model"
$env:GOFAUX_AI_ENDPOINT="http://localhost:1234/v1/chat/completions"
```

The CLI also includes an **AI settings and model selection** menu. It can list local models from:

1. Ollama via the local model list endpoint.
2. OpenAI-compatible local servers via `/v1/models`.
3. The built-in deterministic template provider.

The selected provider and model are saved in `.gofaux.settings.json`. Environment variables can still override the saved settings for experiments.

### Local model file management

GoFaux now includes a model catalog and downloader in `internal/modelhub`. The CLI can download small GGUF model files into `.gofaux/models`. This moves the project toward independence from external desktop tools because model-file management is handled by GoFaux itself.

Managed runner mode:

1. GoFaux can download and manage local model files.
2. GoFaux can install a llama.cpp-compatible local runner into `.gofaux/runners`.
3. GoFaux can start that runner against a selected downloaded GGUF model.
4. GoFaux communicates with the runner through the existing OpenAI-compatible provider.

Thesis sentence:

> To reduce manual setup, the artifact includes a local model catalog, downloader, and managed-runner mode. This allows model files and the local inference process to be managed from inside GoFaux rather than requiring a separately installed model server.

## AI Request DTO

The AI provider receives a structured DTO rather than a loose prompt. This is important for the thesis because it supports repeatable experiments.

Fields include:

1. HTTP method.
2. Endpoint.
3. Status code.
4. Natural-language description.
5. Optional DTO/object name.
6. Optional request DTO/sample JSON.
7. Optional response JSON Schema.
8. Optional explicit field list.

The prompt asks the local model to return only the JSON response body. The result is validated before it becomes a mock.

## Main Engineering Challenges

### Challenge 1: Local AI without vendor lock-in

Hard-coding Ollama would make the implementation simple but would weaken the thesis. A provider interface lets the project test different local runtimes and models while keeping the mock-server core unchanged.

Thesis sentence:

> To avoid coupling the artifact to one model runtime, the assistant module was implemented behind a provider interface. This allows Ollama, LM Studio, llama.cpp, LocalAI, or a deterministic fallback provider to be evaluated under the same mock-generation workflow.

### Challenge 2: LLMs do not always return valid JSON

Even when prompted to return JSON only, language models may produce markdown fences, explanations, trailing text, or malformed objects. The implementation therefore includes JSON extraction and validation before saving generated responses.

Thesis sentence:

> Because structured output from language models is probabilistic, GoFaux 2.0 treats model output as untrusted text. Generated content is accepted only after JSON validation, and invalid output can be passed through a repair prompt.

### Challenge 3: Balancing matching power and simplicity

A fully featured mock server can support many matching rules, but that increases implementation and thesis complexity. GoFaux 2.0 currently focuses on the rules most useful for local development: method, path parameters, query parameters, headers, priority, status, headers, delay, and JSON body.

Thesis sentence:

> The matching model was intentionally limited to the rules most often required during local frontend-backend integration, providing a practical balance between expressiveness, explainability, and implementation scope.

### Challenge 4: Persistent storage versus runtime metrics

Persisting every request counter update would make the configuration file constantly change and introduce disk I/O into runtime serving. The current design keeps request counters in memory during a session.

Thesis sentence:

> Runtime counters are useful for developer feedback, but persisting them after every request would introduce unnecessary disk I/O. Therefore, GoFaux keeps request-count updates in memory while preserving mock definitions in the local configuration file.

### Challenge 5: Usability on normal local machines

The application must work even if no local model is installed. The deterministic template provider exists as a baseline and fallback. This makes development, demonstration, and evaluation possible on machines with limited resources.

Thesis sentence:

> A no-model fallback provider was included to keep the artifact usable on constrained machines and to provide a baseline condition for evaluating AI-assisted generation.

### Challenge 6: Free local AI without embedding a heavy runtime

Running a neural model directly inside the Go binary would require native inference bindings, model-file management, and platform-specific setup. Instead, GoFaux communicates with local model runtimes over HTTP. This keeps the main application simple while still meeting the offline-first and no-paid-API goal.

Thesis sentence:

> GoFaux does not depend on paid cloud AI APIs. AI-assisted authoring is implemented through local provider adapters, allowing the application to use free local runtimes such as Ollama, LM Studio, llama.cpp server, or LocalAI.

Thesis sentence:

> The decision to communicate with a managed local runtime over HTTP avoids implementing neural inference manually in Go, reducing platform-specific complexity while preserving a fully local workflow.

Important thesis nuance:

> Fully local does not necessarily mean that neural inference must be implemented manually in Go. In this thesis, full locality means that prompts, model files, generated outputs, and serving behavior remain on the user's machine. A managed local runner can satisfy this requirement without using paid or cloud APIs.

### Challenge 7: Refactoring from prototype to maintainable architecture

The first implementation mixed CLI interaction, storage, matching, validation, and server behavior across a few small packages. That was acceptable for an MVP but weak for a Master-level software artifact. The refactored structure uses internal modules and explicit dependency wiring.

Thesis sentence:

> During development, the artifact was refactored from a prototype-oriented structure into a modular architecture. This refactoring reduced coupling between components and made it possible to test the mock engine and assistant layer independently.

## Evaluation Ideas Enabled by the Current Design

### Productivity evaluation

Compare manual creation with AI-assisted creation for the same endpoint tasks:

1. `GET /users/{id}`.
2. `GET /products?category=books`.
3. `POST /auth/login`.
4. `GET /orders/{id}`.
5. `POST /payments`.

Metrics:

1. Time to create a working mock.
2. Number of corrections.
3. JSON validity failures.
4. User usefulness rating.
5. User confidence rating.

### JSON generation evaluation

Test several providers/models with the same DTO inputs:

1. Template baseline.
2. Small local instruct model.
3. Medium local instruct model.
4. Code-oriented local model if available.

Metrics:

1. Valid JSON rate.
2. Schema-valid rate if schemas are provided.
3. Repair success rate.
4. Generation latency.
5. Manual edits required.

### Runtime benchmark

Run the mock server with 10, 100, and 1 000 mock definitions.

Metrics:

1. Average latency.
2. p95 latency.
3. Requests per second.
4. Memory usage if feasible.

Expected thesis interpretation:

> Since the AI module is not called during request serving, runtime performance depends mainly on mock matching and HTTP response writing rather than model inference.

## Suggested Next Implementation Steps

1. Add JSON Schema validation for generated responses.
2. Add import/export from OpenAPI examples.
3. Add a more comfortable terminal UI or minimal local web UI.
4. Add benchmark commands or scripts.
5. Add generated evaluation fixtures for 10/100/1 000 mocks.
6. Add a thesis appendix with sample mock configuration and test tasks.
