package main

import (
	"github.com/mangohow/gowlb/cmd/gowlb/internal/addcmd"
	"github.com/mangohow/gowlb/cmd/gowlb/internal/generatecmd"
	"github.com/mangohow/gowlb/cmd/gowlb/internal/projectcmd"
	"github.com/spf13/cobra"
)

var rootCmd = cobra.Command{
	Use:     "mangokit",
	Short:   "mangokit is a toolkit for gin framework service",
	Long:    "mangokit is a toolkit for gin framework service, use proto to define service and error",
	Version: version,
}

func init() {
	rootCmd.AddCommand(projectcmd.CmdProject)
	rootCmd.AddCommand(generatecmd.CmdGenerate)
	rootCmd.AddCommand(addcmd.CmdAdd)
}
