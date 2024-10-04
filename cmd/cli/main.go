package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nullswan/ai/internal/config"
	"github.com/nullswan/ai/internal/ui"

	prompts "github.com/nullswan/ai/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	cfg            *config.Config
	prompt         string
	conversationID string
)

const (
	binName = "golem"
)

var rootCmd = &cobra.Command{
	Use:   binName + " [flags] [arguments]",
	Short: binName + " is an AI runtime",
	Run: func(cmd *cobra.Command, args []string) {
		var selectedPrompt *prompts.Prompt
		if prompt == "" {
			selectedPrompt = &prompts.DefaultPrompts[0]
		} else {
			var err error
			selectedPrompt, err = prompts.LoadPrompt(prompt)
			if err != nil {
				fmt.Printf("Error loading prompt: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Printf("Using prompt: %s\n", selectedPrompt.Name)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// var promptCh chan string
		// if cfg.Input.Text.Enabled {
		// 	fmt.Println("Text input enabled")
		// 	go textinput.Run(
		// 		ctx,
		// 		promptCh,
		// 	)

		// }

		// if cfg.Input.Voice.Enabled {
		// 	fmt.Println("Voice input enabled")
		// 	go voiceinput.Run(
		// 		ctx,
		// 		promptCh,
		// 	)
		// }

		commandCh := make(chan string)

		// Initialize model with channels
		model := ui.NewModel(commandCh)

		program := tea.NewProgram(
			model,
			tea.WithAltScreen(),       // Use the terminal's alternate screen
			tea.WithMouseCellMotion(), // Enable mouse events
		)

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case text := <-commandCh:
					program.Send(ui.NewPagerMsg(text, ui.Human))

					outFile := "output.txt"
					f, err := os.OpenFile(
						outFile,
						os.O_APPEND|os.O_CREATE|os.O_WRONLY,
						0o644,
					)
					if err != nil {
						fmt.Printf("Error opening file: %v\n", err)
						return
					}

					defer f.Close()

					if _, err := f.WriteString(text + "\n"); err != nil {
						fmt.Printf("Error writing to file: %v\n", err)
						return
					}

					go func() {
						time.Sleep(1 * time.Second)

						program.Send(ui.NewPagerMsg("ping", ui.AI))
					}()
				}
			}
		}()

		_, err := program.Run()
		if err != nil {
			os.Exit(1)
		}

		cancel()
	},
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func main() {
	// #region Config commands
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configSetupCmd)
	// #endregion

	// #region Conversation commands
	rootCmd.AddCommand(conversationCmd)
	conversationCmd.AddCommand(conversationListCmd)
	// #endregion

	// #region Output commands
	rootCmd.AddCommand(outputCmd)
	outputCmd.AddCommand(outputListCmd)
	outputCmd.AddCommand(outputAddCmd)
	// #endregion

	// #region Plugin commands
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	// #endregion

	// #region Prompt commands
	rootCmd.AddCommand(promptCmd)
	promptCmd.AddCommand(promptListCmd)
	// #endregion

	// Attach flags to rootCmd only, so they are not inherited by subcommands
	rootCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Specify a prompt")
	rootCmd.Flags().
		StringVarP(&conversationID, "conversation", "c", "", "Specify a conversation ID")

	// Initialize cfg in PersistentPreRun, making it available to all commands
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if !config.ConfigExists() {
			if err := config.Setup(); err != nil {
				fmt.Printf("Error during configuration setup: %v\n", err)
				os.Exit(1)
			}
		}

		var err error
		cfg, err = config.LoadConfig()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}
	}

	// Execute the root command
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}