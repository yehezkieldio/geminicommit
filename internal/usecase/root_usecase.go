package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/fatih/color"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/tfkhdyt/geminicommit/internal/service"
)

type action string

const (
	confirm     action = "CONFIRM"
	regenerate  action = "REGENERATE"
	edit        action = "EDIT"
	editcontext action = "EDIT_CONTEXT"
	cancel      action = "CANCEL"
)

type RootUsecase struct {
	gitService        *service.GitService
	geminiService     *service.GeminiService
	openRouterService *service.OpenRouterService
}

var (
	rootUsecaseInstance *RootUsecase
	rootUsecaseOnce     sync.Once
)

func NewRootUsecase() *RootUsecase {
	rootUsecaseOnce.Do(func() {
		gitService := service.NewGitService()
		geminiService := service.NewGeminiService()
		openRouterService := service.NewOpenRouterService()

		rootUsecaseInstance = &RootUsecase{
			gitService,
			geminiService,
			openRouterService,
		}
	})

	return rootUsecaseInstance
}

func (r *RootUsecase) getRelatedFiles(files []string, quiet *bool) map[string]string {
	relatedFiles := make(map[string]string)
	visitedDirs := make(map[string]bool)

	for idx, file := range files {
		if !*quiet {
			color.New(color.Bold).Printf("     %d. %s\n", idx+1, file)
		}

		dir := filepath.Dir(file)
		if !visitedDirs[dir] {
			lsEntry, err := os.ReadDir(dir)
			if err == nil {
				var ls []string
				for _, entry := range lsEntry {
					ls = append(ls, entry.Name())
				}
				relatedFiles[dir] = strings.Join(ls, ", ")
				visitedDirs[dir] = true
			}
		}
	}

	return relatedFiles
}

func (r *RootUsecase) RootCommand(
	ctx context.Context,
	apiKey string,
	stageAll *bool,
	userContext *string,
	model *string,
	noConfirm *bool,
	quiet *bool,
) error {
	if err := r.gitService.VerifyGitInstallation(); err != nil {
		return err
	}

	if err := r.gitService.VerifyGitRepository(); err != nil {
		return err
	}

	if *stageAll {
		if err := r.gitService.StageAll(); err != nil {
			return err
		}
	}

	filesChan := make(chan []string, 1)
	diffChan := make(chan string, 1)

	if err := spinner.New().
		Title("Detecting staged files").
		Action(func() {
			files, diff, err := r.gitService.DetectDiffChanges()
			if err != nil {
				filesChan <- []string{}
				diffChan <- ""
				return
			}

			filesChan <- files
			diffChan <- diff
		}).
		Run(); err != nil {
		return err
	}

	underline := color.New(color.Underline)
	files, diff := <-filesChan, <-diffChan

	if len(files) == 0 {
		return fmt.Errorf(
			"no staged changes found. stage your changes manually, or automatically stage all changes with the `--all` flag",
		)
	} else if len(files) == 1 && !*quiet {
		underline.Printf("Detected %d staged file:\n", len(files))
	} else if !*quiet {
		underline.Printf("Detected %d staged files:\n", len(files))
	}

	relatedFiles := r.getRelatedFiles(files, quiet)

generate:
	for {
		messageChan := make(chan string, 1)

		if !*quiet {
			if err := spinner.New().
				Title(fmt.Sprintf("AI is analyzing your changes. (Model: %s)", *model)).
				Action(func() {
					r.analyzeToChannel(ctx, apiKey, diff, userContext, relatedFiles, model, messageChan)
				}).
				Run(); err != nil {
				return err
			}
		} else {
			r.analyzeToChannel(ctx, apiKey, diff, userContext, relatedFiles, model, messageChan)
		}

		message := <-messageChan
		if !*quiet {
			underline.Println("\nChanges analyzed!")
		}
		message = strings.TrimSpace(message)

		if message == "" {
			return fmt.Errorf("no commit messages were generated. try again")
		}

		if *noConfirm {
			if err := r.confirmAction(message, quiet); err != nil {
				return err
			}

			return nil
		}

		color.New(color.Bold).Printf("%s\n\n", message)

		var selectedAction action
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[action]().
					Title("Use this commit?").
					Options(
						huh.NewOption("Yes", confirm),
						huh.NewOption("Regenerate", regenerate),
						huh.NewOption("Edit", edit),
						huh.NewOption("Edit Context", editcontext),
						huh.NewOption("Cancel", cancel),
					).
					Value(&selectedAction),
			),
		).Run(); err != nil {
			return err
		}

		switch selectedAction {
		case confirm:
			if err := r.confirmAction(message, quiet); err != nil {
				return err
			}

			break generate
		case regenerate:
			continue
		case edit:
			if err := r.editAction(message); err != nil {
				return err
			}

			break generate
		case editcontext:
			if err := r.editContext(userContext); err != nil {
				return err
			}

			continue
		case cancel:
			color.New(color.FgRed).Println("Commit cancelled")

			break generate
		}
	}

	return nil
}

func (r *RootUsecase) confirmAction(message string, quiet *bool) error {
	if err := r.gitService.CommitChanges(message, quiet); err != nil {
		return err
	}

	if !*quiet {
		color.New(color.FgGreen).Println("✔ Successfully committed!")
	}

	return nil
}

func (r *RootUsecase) editAction(message string) error {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewText().Title("Edit commit message manually").CharLimit(1000).Value(&message),
		),
	).Run(); err != nil {
		return err
	}

	quiet := false

	if err := r.confirmAction(message, &quiet); err != nil {
		return err
	}

	return nil
}

func (r *RootUsecase) editContext(userContext *string) error {
	if err := huh.NewForm(
		huh.NewGroup(
			huh.NewText().Title("Edit user context").CharLimit(1000).Value(userContext),
		),
	).Run(); err != nil {
		return err
	}

	return nil
}

func (r *RootUsecase) analyzeToChannel(
	ctx context.Context,
	apiKey string,
	diff string,
	userContext *string,
	relatedFiles map[string]string,
	model *string,
	messageChan chan string,
) {
	var message string
	var err error

	// Check if the model is a Gemini model
	if strings.HasPrefix(*model, "gemini") {
		client, errClient := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if errClient != nil {
			messageChan <- ""
			return
		}
		defer client.Close()

		message, err = r.geminiService.AnalyzeChanges(
			client,
			ctx,
			diff,
			userContext,
			&relatedFiles,
			model,
		)
	} else {
		// Use OpenRouter for other models
		message, err = r.openRouterService.AnalyzeChanges(
			ctx,
			apiKey,
			diff,
			userContext,
			&relatedFiles,
			model,
		)
	}

	if err != nil {
		messageChan <- ""
	} else {
		messageChan <- message
	}
}
