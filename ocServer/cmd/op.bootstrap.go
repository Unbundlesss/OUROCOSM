//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

/*

{
  "_id": "s3:endlesss-dev.fra1.digitaloceanspaces.com",
  "_rev": "1-4f661e0935d92b271da923bac62b89aa",
  "cdnUrl": "https://endlesss-dev.fra1.cdn.digitaloceanspaces.com",
  "primary": true,
  "type": "S3Endpoint",
  "url": "https://endlesss-dev.fra1.digitaloceanspaces.com"
}
*/

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Configure for first-time server use",
	Long:  `Configure for first-time server use`,
	Run: func(cmd *cobra.Command, args []string) {

		couchClient, err := connectToCouchDB()
		if err != nil {
			SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
		}
		defer couchClient.Close()

	},
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}
