package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Ginboot CLI to the latest version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Updating ginboot-cli...")
		c := exec.Command("go", "install", "github.com/klass-lk/ginboot-cli@latest")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		err := c.Run()
		if err != nil {
			fmt.Printf("Failed to update ginboot-cli: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully updated ginboot-cli to the latest version!")
	},
}
