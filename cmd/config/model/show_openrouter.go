/*
Copyright © 2024 Taufik Hidayat <tfkhdyt@proton.me>
*/
package model

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// showOpenRouterCmd represents the show command for OpenRouter models
var showOpenRouterCmd = &cobra.Command{
	Use:   "show-openrouter",
	Short: "Show currently used OpenRouter model",
	Long:  `Show currently used OpenRouter model`,
	Run: func(cmd *cobra.Command, args []string) {
		model := viper.GetString("api.openrouter_model")
		fmt.Println(model)
	},
}

func init() {
	// Here you will define your flags and configuration settings.
}
