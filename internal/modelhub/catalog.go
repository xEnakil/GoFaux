package modelhub

type ModelSpec struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Family      string `json:"family"`
	Quant       string `json:"quant"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	URL         string `json:"url"`
	Filename    string `json:"filename"`
	LicenseNote string `json:"license_note,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

type DownloadedModel struct {
	Spec      ModelSpec `json:"spec"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
}

func Catalog() []ModelSpec {
	return []ModelSpec{
		{
			ID:          "smollm2-135m-instruct-q4",
			Name:        "SmolLM2 135M Instruct Q4_K_M",
			Family:      "SmolLM2",
			Quant:       "Q4_K_M",
			SizeBytes:   110 * 1024 * 1024,
			URL:         "https://huggingface.co/bartowski/SmolLM2-135M-Instruct-GGUF/resolve/main/SmolLM2-135M-Instruct-Q4_K_M.gguf",
			Filename:    "SmolLM2-135M-Instruct-Q4_K_M.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Very small model for quick offline experiments and low-memory machines.",
		},
		{
			ID:          "qwen2.5-0.5b-instruct-q5",
			Name:        "Qwen2.5 0.5B Instruct Q5_K_M",
			Family:      "Qwen2.5",
			Quant:       "Q5_K_M",
			SizeBytes:   450 * 1024 * 1024,
			URL:         "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q5_k_m.gguf",
			Filename:    "qwen2.5-0.5b-instruct-q5_k_m.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Small general instruct model; useful balance between quality and speed.",
		},
		{
			ID:          "tinyllama-1.1b-chat-q4",
			Name:        "TinyLlama 1.1B Chat Q4_K_M",
			Family:      "TinyLlama",
			Quant:       "Q4_K_M",
			SizeBytes:   670 * 1024 * 1024,
			URL:         "https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf",
			Filename:    "tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Larger than the tiny catalog models but still practical on many laptops.",
		},
		{
			ID:          "qwen2.5-1.5b-instruct-q4",
			Name:        "Qwen2.5 1.5B Instruct Q4_K_M",
			Family:      "Qwen2.5",
			Quant:       "Q4_K_M",
			SizeBytes:   1117320736,
			URL:         "https://huggingface.co/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf",
			Filename:    "qwen2.5-1.5b-instruct-q4_k_m.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Recommended next step for better JSON following while still staying laptop-friendly.",
		},
		{
			ID:          "qwen2.5-3b-instruct-q4",
			Name:        "Qwen2.5 3B Instruct Q4_K_M",
			Family:      "Qwen2.5",
			Quant:       "Q4_K_M",
			SizeBytes:   2104932768,
			URL:         "https://huggingface.co/Qwen/Qwen2.5-3B-Instruct-GGUF/resolve/main/qwen2.5-3b-instruct-q4_k_m.gguf",
			Filename:    "qwen2.5-3b-instruct-q4_k_m.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Heavier but much more useful for structured mock JSON on machines with enough RAM.",
		},
		{
			ID:          "phi3-mini-4k-instruct-q4",
			Name:        "Phi-3 Mini 4K Instruct Q4",
			Family:      "Phi-3",
			Quant:       "Q4",
			SizeBytes:   2393231072,
			URL:         "https://huggingface.co/microsoft/Phi-3-mini-4k-instruct-gguf/resolve/main/Phi-3-mini-4k-instruct-q4.gguf",
			Filename:    "Phi-3-mini-4k-instruct-q4.gguf",
			LicenseNote: "Check the model repository license before redistribution.",
			Notes:       "Alternative stronger instruct model; good candidate when TinyLlama is too weak.",
		},
	}
}

func Find(id string) (ModelSpec, bool) {
	for _, spec := range Catalog() {
		if spec.ID == id {
			return spec, true
		}
	}
	return ModelSpec{}, false
}
