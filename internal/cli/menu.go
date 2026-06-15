package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"GoFaux/internal/assistant"
	"GoFaux/internal/config"
	"GoFaux/internal/httpserver"
	"GoFaux/internal/mock"
	"GoFaux/internal/modelhub"
	"GoFaux/internal/runner"
)

type Menu struct {
	cfg    config.Config
	reader *bufio.Reader
	store  *mock.Store
	server *httpserver.Server
}

func NewMenu(cfg config.Config, store *mock.Store, server *httpserver.Server) *Menu {
	return &Menu{
		cfg:    cfg,
		reader: bufio.NewReader(os.Stdin),
		store:  store,
		server: server,
	}
}

func (m *Menu) Run() error {
	fmt.Println("GoFaux 2.0 - local mock API server")
	fmt.Printf("Config: %s (%d mocks loaded)\n", m.cfg.MockConfigPath, m.store.Count())

	for {
		fmt.Println()
		fmt.Println("1 Add mock manually")
		fmt.Println("2 Generate mock with local AI/provider")
		fmt.Println("3 View mocks")
		fmt.Println("4 Remove mock")
		fmt.Println("5 Start server")
		fmt.Println("6 Stop server")
		fmt.Println("7 AI settings and model selection")
		fmt.Println("8 Local model catalog/downloads")
		fmt.Println("9 Show local settings")
		fmt.Println("10 Exit")
		fmt.Println("------------------------------")

		choice := m.promptLine("Enter your choice: ")
		switch choice {
		case "1":
			m.addManualMock()
		case "2":
			m.addGeneratedMock()
		case "3":
			m.viewMocks()
		case "4":
			m.removeMock()
		case "5":
			m.startServer()
		case "6":
			m.stopServer()
		case "7":
			m.configureAssistant()
		case "8":
			m.manageLocalModels()
		case "9":
			m.showSettings()
		case "10":
			return m.store.Save()
		default:
			fmt.Println("Invalid choice. Try again.")
		}
	}
}

func (m *Menu) addManualMock() {
	fmt.Println()
	fmt.Println("Add mock manually")
	method := m.promptDefault("HTTP method", http.MethodGet)
	endpoint := m.promptDefault("Endpoint path, e.g. /users/{id}", "/users")
	status := m.promptIntDefault("Status code", http.StatusOK)
	name := m.promptLine("Name (optional): ")
	description := m.promptLine("Description (optional): ")
	headers := m.promptKVMap("Response headers, comma separated key=value (optional): ")
	query := m.promptKVMap("Required query params, comma separated key=value (optional): ")
	requestHeaders := m.promptKVMap("Required request headers, comma separated key=value (optional): ")
	delayMS := m.promptIntDefault("Artificial delay in ms", 0)
	priority := m.promptIntDefault("Priority, higher wins", 0)
	body := m.promptJSON("Response JSON body", `{"message":"Hello from GoFaux"}`)

	def, err := m.store.Add(mock.Definition{
		Name:           name,
		Description:    description,
		Method:         method,
		Endpoint:       endpoint,
		Status:         status,
		Headers:        headers,
		Query:          query,
		RequestHeaders: requestHeaders,
		Body:           body,
		DelayMS:        delayMS,
		Priority:       priority,
	})
	if err != nil {
		fmt.Printf("Could not add mock: %v\n", err)
		return
	}
	fmt.Printf("Added %s %s with id %s.\n", def.Method, def.Endpoint, def.ID)
}

func (m *Menu) addGeneratedMock() {
	fmt.Println()
	fmt.Println("Generate mock with local AI/provider")
	fmt.Println("Provider settings: " + m.cfg.Assistant.Description())
	fmt.Println("Tip: write natural language, for example: list people with name and age")

	intent := m.promptDefault("What do you want to mock?", "list people with name and age")
	method, endpoint, description := inferMockRequest(intent)
	method = m.promptDefault("HTTP method", method)
	endpoint = m.promptEndpointDefault("Endpoint path, e.g. /orders/{id}", endpoint)
	status := m.promptIntDefault("Status code", http.StatusOK)
	name := m.promptLine("Mock name (optional): ")
	description = m.promptDefault("Describe the DTO/response you need", description)
	dtoName := m.promptLine("DTO/object name (optional): ")
	fieldHint := m.promptLine("Fields, e.g. id:int,name:string,email:string (optional, Enter to let AI decide): ")
	fields := parseFields(fieldHint)
	if len(fields) == 0 && strings.TrimSpace(fieldHint) != "" {
		description = strings.TrimSpace(description + ". Field hints: " + fieldHint)
	}
	requestDTO := m.promptOptionalJSON("Request DTO/sample JSON to send to the model (optional): ")
	responseSchema := m.promptOptionalJSON("Response JSON Schema to send to the model (optional): ")

	genReq := assistant.GenerateRequest{
		Method:         method,
		Endpoint:       endpoint,
		Status:         status,
		Description:    description,
		DTOName:        dtoName,
		RequestDTO:     requestDTO,
		ResponseSchema: responseSchema,
		Fields:         fields,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	provider, cleanup, err := m.providerForGeneration(ctx)
	if err != nil {
		fmt.Printf("Could not prepare AI provider: %v\n", err)
		return
	}
	defer cleanup()

	result, err := provider.GenerateMock(ctx, genReq)
	if err != nil {
		fmt.Printf("Provider %s failed: %v\n", provider.Name(), err)
		fmt.Println("Falling back to deterministic local template provider.")
		result, err = assistant.TemplateProvider{}.GenerateMock(ctx, genReq)
	}
	if err != nil {
		fmt.Printf("Could not generate mock: %v\n", err)
		return
	}

	body, err := mock.ValidateJSON(string(result.Body))
	if err != nil {
		repaired, repairErr := provider.RepairJSON(ctx, result.Raw, err.Error(), genReq)
		if repairErr == nil {
			body, err = mock.ValidateJSON(string(repaired.Body))
			result = repaired
			result.Attempts++
		}
	}
	if err != nil {
		fmt.Printf("Generated body was not valid JSON: %v\n", err)
		return
	}
	result, body, ok := m.ensureGeneratedBodyQuality(ctx, provider, genReq, result, body)
	if !ok {
		return
	}

	fmt.Println()
	fmt.Println("Generated JSON:")
	fmt.Println(mock.PrettyJSON(body))
	if !m.confirm("Add this mock?", true) {
		fmt.Println("Generation discarded.")
		return
	}

	def, err := m.store.Add(mock.Definition{
		Name:        name,
		Description: description,
		Method:      method,
		Endpoint:    endpoint,
		Status:      status,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: body,
	})
	if err != nil {
		fmt.Printf("Could not add generated mock: %v\n", err)
		return
	}
	fmt.Printf("Added generated mock %s %s with id %s using %s/%s.\n",
		def.Method, def.Endpoint, def.ID, result.Provider, result.Model)
}

func (m *Menu) ensureGeneratedBodyQuality(ctx context.Context, provider assistant.Provider, req assistant.GenerateRequest, result assistant.GenerateResult, body json.RawMessage) (assistant.GenerateResult, json.RawMessage, bool) {
	issues := assistant.ValidateMockQuality(body, req)
	if len(issues) == 0 {
		return result, body, true
	}

	fmt.Println("Generated JSON needs correction:")
	for _, issue := range issues {
		fmt.Printf("- %s\n", issue.Message)
	}

	repaired, repairErr := provider.RepairJSON(ctx, result.Raw, strings.Join(assistant.QualityIssueMessages(issues), "; "), req)
	if repairErr == nil {
		repairedBody, validateErr := mock.ValidateJSON(string(repaired.Body))
		if validateErr == nil {
			repairedIssues := assistant.ValidateMockQuality(repairedBody, req)
			if len(repairedIssues) == 0 {
				fmt.Println("Model corrected the response after a repair prompt.")
				repaired.Attempts = result.Attempts + 1
				return repaired, repairedBody, true
			}
			issues = repairedIssues
		}
	}

	fmt.Println("Using GoFaux local structured fallback for this endpoint.")
	fallback, fallbackErr := assistant.TemplateProvider{}.GenerateMock(ctx, req)
	if fallbackErr != nil {
		fmt.Printf("Could not build fallback mock: %v\n", fallbackErr)
		return result, body, false
	}
	fallbackBody, fallbackErr := mock.ValidateJSON(string(fallback.Body))
	if fallbackErr != nil {
		fmt.Printf("Fallback mock was not valid JSON: %v\n", fallbackErr)
		return result, body, false
	}
	fallbackIssues := assistant.ValidateMockQuality(fallbackBody, req)
	if len(fallbackIssues) > 0 {
		fmt.Println("Fallback mock still did not satisfy quality checks:")
		for _, issue := range fallbackIssues {
			fmt.Printf("- %s\n", issue.Message)
		}
		return result, body, false
	}
	return fallback, fallbackBody, true
}

func (m *Menu) providerForGeneration(ctx context.Context) (assistant.Provider, func(), error) {
	if strings.EqualFold(strings.TrimSpace(m.cfg.Assistant.Provider), "managed") {
		modelPath := strings.TrimSpace(m.cfg.Assistant.Model)
		if modelPath == "" {
			model, ok := m.chooseDownloadedModel()
			if !ok {
				return nil, func() {}, errors.New("no downloaded model selected")
			}
			modelPath = model.Path
		}

		fmt.Println("Starting GoFaux managed local runner...")
		fmt.Println("First run may download the runner into .gofaux/runners.")
		manager := runner.NewManager(m.cfg.RunnerDir)
		running, err := manager.Start(ctx, modelPath)
		if err != nil {
			return nil, func() {}, err
		}
		fmt.Println("Managed runner ready at " + running.Endpoint)
		provider := assistant.NewOpenAICompatibleProvider(running.Endpoint, running.Model, "")
		provider.UseResponseFormat = false
		return provider, running.Stop, nil
	}

	return assistant.NewProvider(m.cfg.Assistant), func() {}, nil
}

func (m *Menu) viewMocks() {
	defs := m.store.All()
	fmt.Println()
	fmt.Println("Mocked APIs")
	fmt.Println("------------------------------")
	if len(defs) == 0 {
		fmt.Println("No mocks added yet.")
		fmt.Println("Use option 2 to generate a mock, for example: list people with name and age.")
		return
	}

	for i, def := range defs {
		fmt.Println(def.Summary(i + 1))
		if def.Description != "" {
			fmt.Printf("    %s\n", def.Description)
		}
		if len(def.Query) > 0 {
			fmt.Printf("    query: %v\n", def.Query)
		}
		if len(def.RequestHeaders) > 0 {
			fmt.Printf("    request headers: %v\n", def.RequestHeaders)
		}
	}
	if m.confirm("View mock details?", false) {
		index := m.promptIntDefault("Mock number", 1)
		if index <= 0 || index > len(defs) {
			fmt.Println("Invalid mock number.")
			return
		}
		m.showMockDetails(defs[index-1], index)
	}
}

func (m *Menu) showMockDetails(def mock.Definition, index int) {
	fmt.Println()
	fmt.Println("Mock details")
	fmt.Println("------------------------------")
	fmt.Printf("Number: %d\n", index)
	fmt.Printf("ID: %s\n", def.ID)
	fmt.Printf("Name: %s\n", withDefault(def.Name, "unnamed"))
	fmt.Printf("Method: %s\n", def.Method)
	fmt.Printf("Endpoint: %s\n", def.Endpoint)
	fmt.Printf("Status: %d\n", def.Status)
	fmt.Printf("Priority: %d\n", def.Priority)
	fmt.Printf("Delay: %d ms\n", def.DelayMS)
	fmt.Printf("Requests received: %d\n", def.Requests)
	if def.Description != "" {
		fmt.Printf("Description: %s\n", def.Description)
	}
	if len(def.Query) > 0 {
		fmt.Printf("Required query: %v\n", def.Query)
	}
	if len(def.RequestHeaders) > 0 {
		fmt.Printf("Required request headers: %v\n", def.RequestHeaders)
	}
	if len(def.Headers) > 0 {
		fmt.Printf("Response headers: %v\n", def.Headers)
	}
	fmt.Println("Response body:")
	fmt.Println(mock.PrettyJSON(def.Body))
}

func (m *Menu) removeMock() {
	m.viewMocks()
	if m.store.Count() == 0 {
		return
	}

	index := m.promptIntDefault("Mock number to remove", 0)
	if index <= 0 {
		fmt.Println("Remove cancelled.")
		return
	}

	removed, ok, err := m.store.DeleteByIndex(index - 1)
	if err != nil {
		fmt.Printf("Could not remove mock: %v\n", err)
		return
	}
	if !ok {
		fmt.Println("Invalid mock number.")
		return
	}
	fmt.Printf("Removed %s %s.\n", removed.Method, removed.Endpoint)
}

func (m *Menu) startServer() {
	if m.store.Count() == 0 {
		fmt.Println("Starting server with 0 mocks. Requests will return 404 until you add or generate a mock.")
	}
	if m.server.Running() {
		fmt.Printf("Server is already running at %s.\n", displayAddr(m.server.Addr()))
		return
	}
	if err := m.server.Start(); err != nil {
		fmt.Printf("Could not start server: %v\n", err)
		return
	}
	fmt.Printf("Server running at %s\n", displayAddr(m.server.Addr()))
	fmt.Println("Health endpoint: " + displayAddr(m.server.Addr()) + "/_gofaux/health")
}

func (m *Menu) stopServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.server.Stop(ctx); err != nil {
		fmt.Printf("Could not stop server: %v\n", err)
		return
	}
	fmt.Println("Server stopped.")
}

func (m *Menu) showSettings() {
	fmt.Println()
	fmt.Println("Local settings")
	fmt.Println("------------------------------")
	fmt.Printf("Config path: %s\n", m.store.ConfigPath())
	fmt.Printf("Settings path: %s\n", m.cfg.SettingsPath)
	fmt.Printf("Model directory: %s\n", m.cfg.ModelDir)
	fmt.Printf("Runner directory: %s\n", m.cfg.RunnerDir)
	fmt.Printf("Mocks loaded: %d\n", m.store.Count())
	fmt.Printf("Server address: %s\n", displayAddr(m.server.Addr()))
	fmt.Printf("Server running: %t\n", m.server.Running())
	fmt.Printf("AI provider: %s\n", m.cfg.Assistant.Description())
}

func (m *Menu) configureAssistant() {
	fmt.Println()
	fmt.Println("AI settings and model selection")
	fmt.Println("------------------------------")
	fmt.Println("1 Template fallback (built in, no model server)")
	fmt.Println("2 Ollama (local, http://localhost:11434)")
	fmt.Println("3 OpenAI-compatible local server (LM Studio, llama.cpp, LocalAI)")
	fmt.Println("4 GoFaux managed runner (downloaded GGUF, no external app)")
	fmt.Println()
	fmt.Println("Current: " + m.cfg.Assistant.Description())

	choice := m.promptDefault("Choose provider", providerChoice(m.cfg.Assistant.Provider))
	next := m.cfg.Assistant
	switch choice {
	case "1":
		next.Provider = "template"
		next.Model = "deterministic-template"
		next.Endpoint = ""
	case "2":
		next.Provider = "ollama"
		next.Endpoint = m.promptDefault("Ollama endpoint", withDefault(next.Endpoint, "http://localhost:11434"))
		next.Model = m.chooseModel(next, "llama3.2:3b")
	case "3":
		next.Provider = "openai"
		next.Endpoint = m.promptDefault("Local chat completions endpoint", withDefault(next.Endpoint, "http://localhost:1234/v1/chat/completions"))
		next.Model = m.chooseModel(next, safeModelDefault(next.Model, "local-model"))
	case "4":
		next.Provider = "managed"
		next.Endpoint = ""
		model, ok := m.chooseDownloadedModel()
		if !ok {
			fmt.Println("No downloaded model selected. Use menu 8 to download a model first.")
			return
		}
		next.Model = model.Path
	default:
		fmt.Println("Unknown provider choice.")
		return
	}

	m.cfg.Assistant = next
	if err := config.SaveSettings(m.cfg); err != nil {
		fmt.Printf("Settings updated for this session, but could not save them: %v\n", err)
		return
	}
	fmt.Println("Saved local AI settings.")
	if strings.EqualFold(m.cfg.Assistant.Provider, "managed") {
		fmt.Println("No mock was created yet. Use option 2 to generate one with this model.")
		if m.confirm("Generate a mock now?", false) {
			m.addGeneratedMock()
		}
	}
}

func (m *Menu) chooseDownloadedModel() (modelhub.DownloadedModel, bool) {
	manager := modelhub.NewManager(m.cfg.ModelDir)
	models, err := manager.ListDownloaded()
	if err != nil {
		fmt.Printf("Could not list downloaded models: %v\n", err)
		return modelhub.DownloadedModel{}, false
	}
	if len(models) == 0 {
		fmt.Println("No downloaded models found. Use menu 8 to download one first.")
		return modelhub.DownloadedModel{}, false
	}

	fmt.Println()
	fmt.Println("Downloaded models:")
	for i, model := range models {
		fmt.Printf("%d %s - %s\n", i+1, model.Spec.Name, formatBytes(model.SizeBytes))
		fmt.Printf("    %s\n", model.Path)
	}
	index := m.promptIntDefault("Model number", 1)
	if index <= 0 || index > len(models) {
		return modelhub.DownloadedModel{}, false
	}
	return models[index-1], true
}

func (m *Menu) manageLocalModels() {
	manager := modelhub.NewManager(m.cfg.ModelDir)

	for {
		fmt.Println()
		fmt.Println("Local model catalog/downloads")
		fmt.Println("------------------------------")
		fmt.Printf("Model directory: %s\n", manager.Dir())
		fmt.Println("1 View catalog")
		fmt.Println("2 Download catalog model")
		fmt.Println("3 List downloaded models")
		fmt.Println("4 Download model from custom URL")
		fmt.Println("5 Back")

		switch m.promptLine("Enter your choice: ") {
		case "1":
			m.showModelCatalog(manager)
		case "2":
			m.downloadCatalogModel(manager)
		case "3":
			m.listDownloadedModels(manager)
		case "4":
			m.downloadCustomModel(manager)
		case "5":
			return
		default:
			fmt.Println("Invalid choice. Try again.")
		}
	}
}

func (m *Menu) showModelCatalog(manager *modelhub.Manager) {
	fmt.Println()
	fmt.Println("Curated local model catalog")
	fmt.Println("------------------------------")
	for i, spec := range manager.Catalog() {
		status := "not downloaded"
		if manager.IsDownloaded(spec) {
			status = "downloaded"
		}
		fmt.Printf("%d %s [%s, %s, approx %s] - %s\n",
			i+1, spec.Name, spec.Family, spec.Quant, formatBytes(spec.SizeBytes), status)
		fmt.Printf("    id: %s\n", spec.ID)
		if spec.Notes != "" {
			fmt.Printf("    %s\n", spec.Notes)
		}
	}
}

func (m *Menu) downloadCatalogModel(manager *modelhub.Manager) {
	catalog := manager.Catalog()
	m.showModelCatalog(manager)
	if len(catalog) == 0 {
		return
	}

	index := m.promptIntDefault("Catalog number to download, or 0 to cancel", 0)
	if index <= 0 {
		fmt.Println("Download cancelled.")
		return
	}
	if index > len(catalog) {
		fmt.Println("Invalid catalog number.")
		return
	}

	m.downloadModel(manager, catalog[index-1])
}

func (m *Menu) downloadCustomModel(manager *modelhub.Manager) {
	url := m.promptLine("Direct GGUF download URL: ")
	if strings.TrimSpace(url) == "" {
		fmt.Println("Download cancelled.")
		return
	}
	filename := m.promptLine("Filename to save as, e.g. my-model.gguf: ")
	if strings.TrimSpace(filename) == "" {
		fmt.Println("Filename is required.")
		return
	}

	spec := modelhub.ModelSpec{
		ID:       "custom-" + strings.TrimSuffix(filename, ".gguf"),
		Name:     filename,
		Family:   "custom",
		Quant:    "unknown",
		URL:      url,
		Filename: filename,
		Notes:    "Custom user-provided model URL.",
	}
	m.downloadModel(manager, spec)
}

func (m *Menu) downloadModel(manager *modelhub.Manager, spec modelhub.ModelSpec) {
	fmt.Println()
	fmt.Printf("Model: %s\n", spec.Name)
	fmt.Printf("Approximate size: %s\n", formatBytes(spec.SizeBytes))
	if spec.LicenseNote != "" {
		fmt.Println(spec.LicenseNote)
	}
	if !m.confirm("Download this model file?", true) {
		fmt.Println("Download cancelled.")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lastPrint := time.Now().Add(-10 * time.Second)
	model, err := manager.Download(ctx, spec, func(progress modelhub.Progress) {
		if time.Since(lastPrint) < time.Second && !progress.Done {
			return
		}
		lastPrint = time.Now()
		if progress.TotalBytes > 0 {
			percent := float64(progress.WrittenBytes) / float64(progress.TotalBytes) * 100
			fmt.Printf("Downloaded %s / %s (%.1f%%)\n",
				formatBytes(progress.WrittenBytes), formatBytes(progress.TotalBytes), percent)
		} else {
			fmt.Printf("Downloaded %s\n", formatBytes(progress.WrittenBytes))
		}
	})
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}
	fmt.Printf("Model saved to %s (%s).\n", model.Path, formatBytes(model.SizeBytes))
	fmt.Println("Use menu 7 and choose GoFaux managed runner to generate mocks with this downloaded model.")
}

func (m *Menu) listDownloadedModels(manager *modelhub.Manager) {
	models, err := manager.ListDownloaded()
	if err != nil {
		fmt.Printf("Could not list downloaded models: %v\n", err)
		return
	}
	if len(models) == 0 {
		fmt.Println("No catalog models downloaded yet.")
		return
	}

	fmt.Println()
	fmt.Println("Downloaded catalog models")
	fmt.Println("------------------------------")
	for i, model := range models {
		fmt.Printf("%d %s - %s\n", i+1, model.Spec.Name, formatBytes(model.SizeBytes))
		fmt.Printf("    %s\n", model.Path)
	}
}

func (m *Menu) chooseModel(cfg assistant.Config, fallback string) string {
	if cfg.Provider == "template" {
		return "deterministic-template"
	}

	if m.confirm("Discover local models from the provider?", true) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		models, err := assistant.ListModels(ctx, cfg)
		cancel()
		if err != nil {
			fmt.Printf("Could not list local models: %v\n", err)
		} else if len(models) == 0 {
			fmt.Println("No local models reported by the provider.")
		} else {
			fmt.Println()
			fmt.Println("Available local models:")
			for i, model := range models {
				size := ""
				if model.Size > 0 {
					size = fmt.Sprintf(" (%s)", formatBytes(model.Size))
				}
				fmt.Printf("%d %s%s\n", i+1, model.Name, size)
			}
			index := m.promptIntDefault("Model number, or 0 to type manually", 1)
			if index > 0 && index <= len(models) {
				return models[index-1].Name
			}
		}
	}

	return m.promptDefault("Model name", withDefault(cfg.Model, fallback))
}

func (m *Menu) promptLine(label string) string {
	fmt.Print(label)
	text, _ := m.reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func (m *Menu) promptDefault(label string, fallback string) string {
	value := m.promptLine(fmt.Sprintf("%s [%s]: ", label, fallback))
	if value == "" {
		return fallback
	}
	return value
}

func (m *Menu) promptEndpointDefault(label string, fallback string) string {
	value := m.promptLine(fmt.Sprintf("%s [%s]: ", label, fallback))
	if value == "" {
		return fallback
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	switch lower {
	case "list", "get", "show", "fetch":
		fmt.Printf("Using inferred endpoint %s instead of action word %q.\n", fallback, value)
		return fallback
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return value
}

func (m *Menu) promptIntDefault(label string, fallback int) int {
	for {
		value := m.promptLine(fmt.Sprintf("%s [%d]: ", label, fallback))
		if value == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
		fmt.Println("Enter a valid number.")
	}
}

func (m *Menu) promptKVMap(label string) map[string]string {
	return parseKVMap(m.promptLine(label))
}

func (m *Menu) promptJSON(label string, fallback string) json.RawMessage {
	for {
		value := m.promptDefault(label, fallback)
		raw, err := mock.ValidateJSON(value)
		if err == nil {
			return raw
		}
		fmt.Printf("Invalid JSON: %v\n", err)
	}
}

func (m *Menu) promptOptionalJSON(label string) json.RawMessage {
	for {
		value := m.promptLine(label)
		if value == "" {
			return nil
		}
		raw, err := mock.ValidateJSON(value)
		if err == nil {
			return raw
		}
		fmt.Printf("Invalid JSON: %v\n", err)
	}
}

func (m *Menu) confirm(label string, fallback bool) bool {
	defaultLabel := "Y/n"
	if !fallback {
		defaultLabel = "y/N"
	}
	value := strings.ToLower(m.promptLine(fmt.Sprintf("%s [%s]: ", label, defaultLabel)))
	if value == "" {
		return fallback
	}
	return value == "y" || value == "yes"
}

func parseKVMap(line string) map[string]string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	out := map[string]string{}
	for _, part := range strings.Split(line, ",") {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		key, value, ok := strings.Cut(piece, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseFields(line string) []assistant.FieldSpec {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	if !strings.Contains(line, ":") {
		return parseFieldNamesOnly(line)
	}

	var fields []assistant.FieldSpec
	for _, part := range strings.Split(line, ",") {
		piece := strings.TrimSpace(part)
		if piece == "" {
			continue
		}
		name := piece
		typeHint := ""
		if left, right, ok := strings.Cut(piece, ":"); ok {
			name = strings.TrimSpace(left)
			typeHint = strings.TrimSpace(right)
		}
		if name != "" {
			fields = append(fields, assistant.FieldSpec{Name: name, Type: typeHint, Required: true})
		}
	}
	return fields
}

func parseFieldNamesOnly(line string) []assistant.FieldSpec {
	var fields []assistant.FieldSpec
	for _, part := range strings.Split(line, ",") {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if !isSimpleFieldName(name) {
			return nil
		}
		fields = append(fields, assistant.FieldSpec{Name: name, Required: true})
	}
	return fields
}

func isSimpleFieldName(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func displayAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	if strings.HasPrefix(addr, ":") {
		return "http://localhost" + addr
	}
	return "http://" + addr
}

func providerChoice(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "ollama":
		return "2"
	case "openai", "lmstudio", "llamacpp", "localai":
		return "3"
	case "managed":
		return "4"
	default:
		return "1"
	}
}

func withDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func safeModelDefault(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasSuffix(strings.ToLower(value), ".gguf") {
		return fallback
	}
	return value
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func inferMockRequest(input string) (method, endpoint, description string) {
	cleaned := strings.TrimSpace(input)
	if cleaned == "" {
		return http.MethodGet, "/items", "A realistic JSON response for local development"
	}

	lower := strings.ToLower(cleaned)
	method = http.MethodGet
	switch {
	case strings.HasPrefix(lower, "create "), strings.HasPrefix(lower, "add "), strings.HasPrefix(lower, "post "):
		method = http.MethodPost
	case strings.HasPrefix(lower, "update "), strings.HasPrefix(lower, "put "):
		method = http.MethodPut
	case strings.HasPrefix(lower, "delete "), strings.HasPrefix(lower, "remove "):
		method = http.MethodDelete
	}

	resource := "items"
	words := strings.Fields(lower)
	for i, word := range words {
		word = strings.Trim(word, ".,:;!?")
		if word == "list" && i+1 < len(words) {
			for j := i + 1; j < len(words); j++ {
				candidate := strings.Trim(words[j], ".,:;!?")
				if isResourceFillerWord(candidate) {
					continue
				}
				resource = sanitizePathWord(candidate)
				break
			}
			if resource != "items" {
				break
			}
		}
		switch word {
		case "people", "persons":
			resource = "people"
		case "users", "user":
			resource = "users"
		case "products", "product":
			resource = "products"
		case "orders", "order":
			resource = "orders"
		}
		if resource != "items" {
			break
		}
	}

	return method, "/" + resource, cleaned
}

func isResourceFillerWord(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "of", "the", "a", "an", "all", "some", "many", "maybe", "with", "and", "like":
		return true
	default:
		return false
	}
}

func sanitizePathWord(value string) string {
	value = strings.ToLower(strings.Trim(value, " /.,:;!?"))
	value = strings.ReplaceAll(value, "_", "-")
	if value == "person" || value == "persons" {
		return "people"
	}
	if value == "" {
		return "items"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "items"
	}
	return b.String()
}
