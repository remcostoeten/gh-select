package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/atotto/clipboard"
	"github.com/remcostoeten/gh-select/internal/tui"
	"github.com/spf13/cobra"
)

var (
	noCache bool

	rootCmd = &cobra.Command{
		Use:   "gh-select",
		Short: "Interactive GitHub Repository Selector",
		Long: `Fuzzy-find and manage your GitHub repos from the terminal.
A GitHub CLI extension following the gh extension spec.`,
		Run: func(cmd *cobra.Command, args []string) {
			repo, err := tui.Start(noCache)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			if repo == nil {
				fmt.Println("No repository selected")
				return
			}

			action, err := tui.SelectAction(repo)
			if err != nil {
				fmt.Println("Error:", err)
				os.Exit(1)
			}

			switch action {
			case "clone":
				fmt.Printf("\nCloning %s...\n", repo.NameWithOwner)
				cmd := exec.Command("gh", "repo", "clone", repo.NameWithOwner)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Run()
			case "copy-name":
				if err := clipboard.WriteAll(repo.NameWithOwner); err != nil {
					fmt.Printf("Failed to copy to clipboard: %v\n", err)
				} else {
					fmt.Printf("\nCopied %s to clipboard!\n", repo.NameWithOwner)
				}
			case "copy-url":
				url := fmt.Sprintf("https://github.com/%s", repo.NameWithOwner)
				if err := clipboard.WriteAll(url); err != nil {
					fmt.Printf("Failed to copy to clipboard: %v\n", err)
				} else {
					fmt.Printf("\nCopied %s to clipboard!\n", url)
				}
			case "open":
				fmt.Printf("\nOpening in browser...\n")
				cmd := exec.Command("gh", "repo", "view", repo.NameWithOwner, "--web")
				cmd.Run()
			case "print":
				fmt.Println(repo.NameWithOwner)
			case "":
				fmt.Println("No action selected")
			default:
				fmt.Println("Invalid choice")
			}
		},
	}
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&noCache, "no-cache", "n", false, "Disable cache")
}
