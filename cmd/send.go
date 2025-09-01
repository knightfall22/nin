/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/knightfall22/nin/transmission"
	"github.com/spf13/cobra"
)

// sendCmd represents the send command
var sendCmd = &cobra.Command{
	Use:          "send <path>",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	Aliases:      []string{"s"},
	Short:        "Send a file to listners",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug, err := cmd.Flags().GetInt("debug")
		if err != nil {
			return err
		}

		transmission.Debug = debug

		zip, err := cmd.Flags().GetString("zip")
		if err != nil {
			return err
		}

		zipDelete, err := cmd.Flags().GetBool("zipdelete")
		if err != nil {
			return err
		}

		multicast, err := cmd.Flags().GetString("multicast")
		if err != nil {
			return err
		}

		listners, err := cmd.Flags().GetInt("listners")
		if err != nil {
			return err
		}

		delay, err := cmd.Flags().GetDuration("delay")
		if err != nil {
			return err
		}

		fmt.Println("delay", delay)

		p := new(transmission.Peer)
		opts := transmission.Options{
			FilePath:               args[0],
			ZipFolder:              zip,
			ZipDeleteComplete:      zipDelete,
			MulticastAddress:       multicast,
			ListenerLimit:          listners,
			AutomaticShutdownDelay: delay,
		}

		err = p.Send(opts)
		return err
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	sendCmd.PersistentFlags().String("zip", "", "zip folder path")
	sendCmd.PersistentFlags().Bool("zipdelete", true, "delete zip folder after sending(default=true)")
	sendCmd.PersistentFlags().String("multicast", "", "multicast address")
	sendCmd.PersistentFlags().Int("listners", 0, "number of listners(default=4)")
	sendCmd.PersistentFlags().Duration("delay", transmission.DefaultAutomaticShutdownDelay, "automatic shutdown delay(default=60s)")
	sendCmd.PersistentFlags().Int("debug", 0, "debug level(default=0)")
	// sendCmd.PersistentFlags().Int("retries", 0, "max piece retries(default=0)")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sendCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
