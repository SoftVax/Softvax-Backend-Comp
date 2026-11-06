package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	defaultModel   = "gpt-4.1-mini"
	openAIEndpoint = "https://api.openai.com/v1/responses"
	maxBodyBytes   = 1 << 20
)

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Response string `json:"response"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type openAIResponse struct {
	Output []struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultModel
	}

	client := &http.Client{Timeout: 60 * time.Second}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /chat", chatHandler(client, apiKey, model))

	address := ":" + envOrDefault("PORT", "8080")
	log.Printf("server listening on %s", address)
	server := &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      65 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func chatHandler(client *http.Client, apiKey, model string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input chatRequest
		if err := decodeJSON(w, r, &input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		input.Message = strings.TrimSpace(input.Message)
		if input.Message == "" {
			writeError(w, http.StatusBadRequest, "message is required")
			return
		}

		payload, err := json.Marshal(map[string]string{
			"model": model,
			"input": input.Message,
		})
		if err != nil {
			log.Printf("encode OpenAI request: %v", err)
			writeError(w, http.StatusInternalServerError, "could not prepare AI request")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, openAIEndpoint, bytes.NewReader(payload))
		if err != nil {
			log.Printf("create OpenAI request: %v", err)
			writeError(w, http.StatusInternalServerError, "could not prepare AI request")
			return
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("call OpenAI API: %v", err)
			writeError(w, http.StatusBadGateway, "AI service is unavailable")
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
		if err != nil {
			log.Printf("read OpenAI response: %v", err)
			writeError(w, http.StatusBadGateway, "could not read AI response")
			return
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			log.Printf("OpenAI API returned status %d", resp.StatusCode)
			writeError(w, http.StatusBadGateway, "AI service returned an error")
			return
		}

		var result openAIResponse
		if err := json.Unmarshal(body, &result); err != nil {
			log.Printf("decode OpenAI response: %v", err)
			writeError(w, http.StatusBadGateway, "invalid response from AI service")
			return
		}

		text := extractOutputText(result)
		if text == "" {
			log.Printf("OpenAI response contained no output text")
			writeError(w, http.StatusBadGateway, "AI service returned no text")
			return
		}

		writeJSON(w, http.StatusOK, chatResponse{Response: text})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body must contain one JSON value")
		}
		return err
	}
	return nil
}

func extractOutputText(result openAIResponse) string {
	var parts []string
	for _, item := range result.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" && content.Text != "" {
				parts = append(parts, content.Text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
