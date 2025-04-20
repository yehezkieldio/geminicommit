package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

type OpenRouterService struct {
	systemPrompt string
}

var (
	openRouterService *OpenRouterService
	openRouterOnce    sync.Once
)

func NewOpenRouterService() *OpenRouterService {
	openRouterOnce.Do(func() {
		openRouterService = &OpenRouterService{
			systemPrompt: `You are a commit message generator that follows these rules:
1. Write in first-person singular present tense
2. Be concise and direct
3. Output only the commit message without any explanations
4. Follow the format: <type>(<optional scope>): <commit message>
5. Commit message should starts with lowercase letter.
6. Commit message must be a maximum of 72 characters.
7. Exclude anything unnecessary such as translation. Your entire response will be passed directly into git commit.`,
		}
	})

	return openRouterService
}

func (g *OpenRouterService) GetUserPrompt(
	context *string,
	diff string,
	files []string,
) (string, error) {
	if *context != "" {
		temp := fmt.Sprintf("Use the following context to understand intent: %s", *context)
		context = &temp
	} else {
		*context = ""
	}

	conventionalTypes, err := json.Marshal(map[string]string{
		"docs":     "Documentation only changes",
		"style":    "Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)",
		"refactor": "A code change that neither fixes a bug nor adds a feature",
		"perf":     "A code change that improves performance",
		"test":     "Adding missing tests or correcting existing tests",
		"build":    "Changes that affect the build system or external dependencies",
		"ci":       "Changes to our CI configuration files and scripts",
		"chore":    "Other changes that don't modify src or test files",
		"revert":   "Reverts a previous commit",
		"feat":     "A new feature",
		"fix":      "A bug fix",
	})
	if err != nil {
		return "", fmt.Errorf("error marshaling conventional types: %v", err)
	}

	return fmt.Sprintf(
		`Generate a concise git commit message written in present tense for the following code diff with the given specifications below:

The output response must be in format:
<type>(<optional scope>): <commit message>

%s

Choose a type from the type-to-description JSON below that best describes the git diff:
%s

Neighboring files:
%s

Code diff:
%s`,
		*context,
		conventionalTypes,
		strings.Join(files, ", "),
		diff,
	), nil
}

func (g *OpenRouterService) AnalyzeChanges(
	ctx context.Context,
	apiKey string,
	diff string,
	userContext *string,
	relatedFiles *map[string]string,
	modelName *string,
) (string, error) {
	relatedFilesArray := make([]string, 0, len(*relatedFiles))
	for dir, ls := range *relatedFiles {
		relatedFilesArray = append(relatedFilesArray, fmt.Sprintf("%s/%s", dir, ls))
	}

	httpClient := &http.Client{
		Transport: &customTransport{
			headers: map[string]string{
				"HTTP-Referer": "https://github.com/yehezkieldio/geminicommit",
				"X-Title":      "forked-geminicommit",
			},
			transport: http.DefaultTransport,
		},
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	config.HTTPClient = httpClient

	client := openai.NewClientWithConfig(config)

	userPrompt, err := g.GetUserPrompt(userContext, diff, relatedFilesArray)
	if err != nil {
		return "", err
	}

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: *modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: g.systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			MaxTokens: 100,
		},
	)

	if err != nil {
		return "", fmt.Errorf("error creating chat completion: %v", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

type customTransport struct {
	headers   map[string]string
	transport http.RoundTripper
}

func (ct *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	for key, value := range ct.headers {
		req.Header.Add(key, value)
	}

	return ct.transport.RoundTrip(req)
}
