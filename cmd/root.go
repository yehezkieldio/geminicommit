/*
Copyright © 2024 Taufik Hidayat <tfkhdyt@proton.me>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/yehezkieldio/geminicommit/cmd/config"
	"github.com/yehezkieldio/geminicommit/internal/delivery/cli/handler"
)

var (
	cfgFile     string
	stageAll    = false
	userContext string
	model       string
	noConfirm   = false
	quiet       = false
	rootHandler = handler.NewRootHandler()
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "geminicommit",
	Short:   "CLI that writes your git commit messages for you with Google Gemini AI",
	Long:    "CLI that writes your git commit messages for you with Google Gemini AI",
	Version: "0.2.6",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: rootHandler.RootCommand(
		context.Background(),
		&stageAll,
		&userContext,
		&model,
		&noConfirm,
		&quiet,
	),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.AddCommand(config.ConfigCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/geminicommit/config.toml)")
	RootCmd.Flags().
		BoolVarP(&stageAll, "all", "a", stageAll, "stage all changes in tracked files")
	RootCmd.Flags().
		BoolVarP(&noConfirm, "yes", "y", noConfirm, "skip confirmation prompt")
	RootCmd.Flags().
		BoolVarP(&quiet, "quiet", "q", quiet, "suppress output (only works with --yes)")
	RootCmd.Flags().
		StringVarP(&userContext, "context", "c", "", "additional context to be added to the commit message")
	RootCmd.Flags().
		StringVarP(&model, "model", "m", "gemini-1.5-pro", "model to use (gemini or openrouter models)")

	RootCmd.RegisterFlagCompletionFunc("model", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"gemini-1.5-pro",
			"openai/gpt-4",
			"openai/gpt-3.5-turbo",
			"anthropic/claude-2",
			"anthropic/claude-instant-1",
			"google/palm-2-chat-bison",
			"google/gemini-pro",
			"meta-llama/llama-2-70b-chat",
			"meta-llama/llama-2-13b-chat",
			"mistral/mistral-7b-instruct",
		}, cobra.ShellCompDirectiveNoFileComp
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find config directory.
		config, err := os.UserConfigDir()
		cobra.CheckErr(err)
		configDirPath := filepath.Join(config, "geminicommit")
		configFilePath := filepath.Join(configDirPath, "config.toml")

		viper.AddConfigPath(configDirPath)
		viper.SetConfigType("toml")
		viper.SetConfigName("config")

		if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
			createConfig()
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Error: failed to read config")
		os.Exit(1)
	}
}

func createConfig() {
	// Create the directory and file paths.
	config, err := os.UserConfigDir()
	cobra.CheckErr(err)
	configDirPath := filepath.Join(config, "geminicommit")
	configFilePath := filepath.Join(configDirPath, "config.toml")

	// Create the directory if it does not exist.
	if _, err := os.Stat(configDirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configDirPath, 0o755); err != nil {
			fmt.Println("Error: failed to make config dir")
			os.Exit(1)
		}
	}

	// Create the file if it does not exist.
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		file, err := os.Create(configFilePath)
		if err != nil {
			fmt.Println("Error: failed to make config file")
			os.Exit(1)
		}
		defer file.Close()
	}
}
