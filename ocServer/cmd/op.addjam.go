//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cmdJamName = ""

var addjamCmd = &cobra.Command{
	Use:   "addjam",
	Short: "Create a new jam database",
	Long:  `Create a new jam database`,
	Run: func(cmd *cobra.Command, args []string) {

		couchClient, err := connectToCouchDB()
		if err != nil {
			SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
		}
		defer couchClient.Close()

		err = createDefaultPublicJamDatabase(couchClient, cmdJamName)
		if err != nil {
			SysLog.Fatal("Failed to create new jam database", zap.String("Jam", cmdJamName), zap.Error(err))
		}

		SysLog.Info("Created new jam database successfully", zap.String("Jam", cmdJamName))
	},
}

func init() {
	rootCmd.AddCommand(addjamCmd)

	addjamCmd.Flags().StringVarP(&cmdJamName, "name", "n", "", "Name of the new jam to add")
	addjamCmd.MarkFlagRequired("name")
}
