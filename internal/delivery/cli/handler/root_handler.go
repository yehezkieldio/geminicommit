package handler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yehezkieldio/geminicommit/internal/usecase"
)

type RootHandler struct {
	useCase *usecase.RootUsecase
}

var (
	rootHandlerInstance *RootHandler
	rootHandlerOnce     sync.Once
)

func NewRootHandler() *RootHandler {
	rootHandlerOnce.Do(func() {
		useCase := usecase.NewRootUsecase()

		rootHandlerInstance = &RootHandler{useCase}
	})

	return rootHandlerInstance
}

func (r *RootHandler) RootCommand(
	ctx context.Context,
	stageAll *bool,
	userContext *string,
	model *string,
	noConfirm *bool,
	quiet *bool,
) func(*cobra.Command, []string) {
	return func(_ *cobra.Command, _ []string) {
		modelFromConfig := viper.GetString("api.model")
		if modelFromConfig != "" && *model == "gemini-1.5-pro" {
			*model = modelFromConfig
		}

		if *quiet && !*noConfirm {
			*quiet = false
		}

		// Get appropriate API key based on model
		var apiKey string
		if strings.HasPrefix(*model, "gemini") {
			apiKey = viper.GetString("api.key")
			if apiKey == "" {
				fmt.Println(
					"Error: Gemini API key is still empty, run this command to set your API key",
				)
				fmt.Print("\n")
				color.New(color.Bold).Print("geminicommit config key set ")
				color.New(color.Italic, color.Bold).Print("api_key\n\n")
				os.Exit(1)
			}
		} else {
			apiKey = viper.GetString("api.openrouter_key")
			if apiKey == "" {
				fmt.Println(
					"Error: OpenRouter API key is still empty, run this command to set your API key",
				)
				fmt.Print("\n")
				color.New(color.Bold).Print("geminicommit config key set-openrouter ")
				color.New(color.Italic, color.Bold).Print("api_key\n\n")
				os.Exit(1)
			}
		}

		err := r.useCase.RootCommand(ctx, apiKey, stageAll, userContext, model, noConfirm, quiet)
		cobra.CheckErr(err)
	}
}
