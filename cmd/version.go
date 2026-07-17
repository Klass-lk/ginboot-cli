package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Ginboot CLI",
	Run: func(cmd *cobra.Command, args []string) {
		info, ok := debug.ReadBuildInfo()
		if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
			fmt.Printf("ginboot-cli version %s\n", info.Main.Version)
		} else {
			fmt.Println("ginboot-cli version (development)")
		}
	},
}
