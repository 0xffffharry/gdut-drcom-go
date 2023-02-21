package main

import "github.com/spf13/cobra"

var RootCommand = &cobra.Command{
	Use:   "gdut-drcom-go",
	Short: "A golang drcom client for GDUT",
}

func main() {
	RootCommand.Execute()
}
