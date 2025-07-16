//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	kivik "github.com/go-kivik/kivik/v4"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	zip2 "github.com/kdungs/zip"
)

var (
	cmdOutputDir          = "."
	cmdJamToExport        = ""
	cmdServerNamePrefix   = ""
	cmdStemS3Server       = ""
	cmdIgnoreMissingStems = false
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a jam to LORE jam archival format",
	Long:  `Export a jam to LORE jam archival format`,
	Run: func(cmd *cobra.Command, args []string) {
		exportJamToDisk(cmdOutputDir, cmdJamToExport, cmdServerNamePrefix, cmdStemS3Server, cmdIgnoreMissingStems)
	},
}

type UserExportData struct {
	ID        string `json:"_id"`
	UserName  string `json:"name"`
	LoginPass string `json:"login"`
}

func compressWithPassword(inputFiles []string, password, outputZipPath string) error {

	fo, err := os.Create(outputZipPath)
	if err != nil {
		return err
	}
	defer func() {
		if err := fo.Close(); err != nil {
			SysLog.Warn("Failed to close zip file", zap.Error(err))
		}
	}()
	fileOut := bufio.NewWriter(fo)
	fileOutZip := zip2.NewWriter(fileOut)
	defer fileOutZip.Close()

	for _, filePath := range inputFiles {
		filenameRelative := filepath.Base(filePath)

		fileContents, err := os.ReadFile(filePath)
		if err != nil {
			return errors.Join(fmt.Errorf("unable to load [%s] for zip file", filePath), err)
		}

		w, err := fileOutZip.Encrypt(filenameRelative, password, zip2.AES256Encryption)
		if err != nil {
			return errors.Join(fmt.Errorf("failed to create encrypted entry [%s] in zip file", filenameRelative), err)
		}
		_, err = io.Copy(w, bytes.NewReader(fileContents))
		if err != nil {
			return errors.Join(fmt.Errorf("failed to encrypt [%s] into zip file", filenameRelative), err)
		}
	}
	return nil
}

var exportSolosCmd = &cobra.Command{
	Use:   "exportsolo",
	Short: "Export all solo jams to compressed LORE archives",
	Long:  `Export all solo jams to compressed LORE archives`,
	Run: func(cmd *cobra.Command, args []string) {

		couchClient, err := connectToCouchDB()
		if err != nil {
			SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
		}
		defer couchClient.Close()

		// create a special output root for all the encrypted results
		soloEncFileRoot := path.Join(cmdOutputDir, "_solos")
		os.MkdirAll(soloEncFileRoot, os.ModePerm)

		userDb := couchClient.DB("_users")
		resultSet := userDb.AllDocs(context.TODO(), kivik.Params(map[string]interface{}{
			"include_docs": true,
		}))
		defer resultSet.Close()

		// walk the full list of users
		for resultSet.Next() {
			var doc UserExportData
			if err := resultSet.ScanDoc(&doc); err != nil {
				SysLog.Error("[ExportSolo] ResultSet ScanDoc failure", zap.Error(err))
			} else {
				// grab the solo data from Couch and S3, should produce a .yaml and .tar file
				generatedFiles, err := exportJamToDisk(cmdOutputDir, doc.UserName, cmdServerNamePrefix, cmdStemS3Server, cmdIgnoreMissingStems)
				if err != nil {
					SysLog.Error("[ExportSolo] Export process failed", zap.Error(err))
				}

				if len(generatedFiles) >= 2 {

					encFilePath := path.Join(soloEncFileRoot, fmt.Sprintf("%s.solo_encrypted.zip", doc.UserName))

					// compress those .yaml and .tar files into an encrypted .zip with the users' password
					// so it's easy to archive these but with enough protection to stop idle snooping
					err = compressWithPassword(generatedFiles, doc.LoginPass, encFilePath)
					if err != nil {
						SysLog.Error("[ExportSolo] Compression failed", zap.Error(err))
					} else {
						SysLog.Info("Exporting user ["+doc.UserName+"]", zap.Strings("Files", generatedFiles))
					}
				} else {
					SysLog.Warn("Ignoring user [" + doc.UserName + "] - no files to compress")
				}
			}
		}
		if resultSet.Err() != nil {
			SysLog.Error("[ExportSolo] ResultSet general failure", zap.Error(resultSet.Err()))
		}
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(exportSolosCmd)

	// tool for exporting a single jam by ID
	{
		exportCmd.Flags().StringVarP(&cmdOutputDir, "out", "o", "", "output directory to write to / use as cache root")

		exportCmd.Flags().StringVarP(&cmdJamToExport, "jam", "j", "", "(required) COSMID jam ID to export")
		exportCmd.MarkFlagRequired("jam")
		exportCmd.Flags().StringVarP(&cmdServerNamePrefix, "prefix", "p", "", "(required) Server name prefix applied to each jam export")
		exportCmd.MarkFlagRequired("prefix")
		exportCmd.Flags().StringVarP(&cmdStemS3Server, "stem", "s", "", "if given, talk to this S3 server to fetch the stems and bake them into a .tar")

		exportCmd.Flags().BoolVarP(&cmdIgnoreMissingStems, "ignore-missing", "i", false, "ignore any 404 responses when downloading stem data")
	}
	// export tool for all solo jams at once, with per-archive encryption
	{
		exportSolosCmd.Flags().StringVarP(&cmdOutputDir, "out", "o", "", "output directory to write to / use as cache root")

		exportSolosCmd.Flags().StringVarP(&cmdServerNamePrefix, "prefix", "p", "", "(required) Server name prefix applied to each jam export")
		exportSolosCmd.MarkFlagRequired("prefix")
		exportSolosCmd.Flags().StringVarP(&cmdStemS3Server, "stem", "s", "", "if given, talk to this S3 server to fetch the stems and bake them into a .tar")

		exportSolosCmd.Flags().BoolVarP(&cmdIgnoreMissingStems, "ignore-missing", "i", false, "ignore any 404 responses when downloading stem data")
	}
}
