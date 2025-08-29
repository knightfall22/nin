/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/knightfall22/nin/transmission"
	"github.com/spf13/cobra"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:          "listen",
	SilenceUsage: true,
	Aliases:      []string{"s"},
	Short:        "A brief description of your command",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, err := cmd.Flags().GetInt("debug")
		if err != nil {
			return err
		}

		transmission.Debug = debug

		senderAddr, err := cmd.Flags().GetString("sender")
		if err != nil {
			return err
		}

		retries, err := cmd.Flags().GetInt("maxretry")
		if err != nil {
			return err
		}

		path, err := cmd.Flags().GetString("path")
		if err != nil {
			return err
		}

		l := new(transmission.Peer)
		err = l.Listen(transmission.Options{
			DownloadFilePath: path,
			MaxPieceRetries:  retries,
			SenderAddress:    senderAddr,
		})

		return err
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	listenCmd.PersistentFlags().String("sender", "", "Address of the sender")
	listenCmd.PersistentFlags().Int("maxretry", 4, "Amount of retires of a piece before it download cancels")
	listenCmd.PersistentFlags().String("path", "", "path to store the files")
	listenCmd.PersistentFlags().Int("debug", 0, "debug level(default=0)")
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
