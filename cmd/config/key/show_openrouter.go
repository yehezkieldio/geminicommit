/*
Copyright © 2024 Taufik Hidayat <tfkhdyt@proton.me>
*/
package key

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// showOpenRouterCmd represents the show command
var showOpenRouterCmd = &cobra.Command{
	Use:   "show-openrouter",
	Short: "Show currently used OpenRouter API key",
	Long:  `Show currently used OpenRouter API key`,
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := viper.GetString("api.openrouter_key")
		fmt.Println(apiKey)
	},
}

func init() {
	// Here you will define your flags and configuration settings.
}
