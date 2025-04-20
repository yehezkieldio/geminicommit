/*
Copyright © 2024 Taufik Hidayat <tfkhdyt@proton.me>
*/
package model

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// setOpenRouterCmd represents the set command for OpenRouter models
var setOpenRouterCmd = &cobra.Command{
	Use:   "set-openrouter {model_name}",
	Short: "Set OpenRouter model",
	Long:  `Set OpenRouter model to be used for generating commit messages.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		model := args[0]
		viper.Set("api.openrouter_model", model)
		cobra.CheckErr(viper.WriteConfig())
	},
}

func init() {
	// Here you will define your flags and configuration settings.
}
