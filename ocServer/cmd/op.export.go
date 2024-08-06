//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cmdJamToExport = ""
var cmdServerNamePrefix = ""
var cmdStemS3Server = ""

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export a jam to LORE jam archival format",
	Long:  `Export a jam to LORE jam archival format`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(cmdJamToExport) == 0 {
			SysLog.Fatal("Jam to export cannot be null")
		}

		// when yanking stems from our server, we might be pointing at localhost or similar which
		// (as it has to be over https) would kick up cert issues, so skip verification by default
		httpTransport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := &http.Client{Transport: httpTransport}

		// lookup couch ID for access
		idBank := SysBankIDs.Bank()
		lutID, ok := idBank.Entries[cmdJamToExport]
		if !ok {
			SysLog.Fatal("Unable to resolve COSMID to Endlesss jam IDs", zap.String("COSMID", cmdJamToExport))
		}

		// build a custom band ID for our import; we re-use endlesss IDs currently so we don't want them
		// conflicting with existing LORE data. instead, we take the fourcc identifier for the server and
		// the jam_### COSMID and crunch it down into a new unique-to-the-server band ID
		// .. it's not hex digits, but LORE doesn't care about that
		// eg
		// band + csmx + jam001 = bandcsmxjam001
		//                        bandb9eb4938e8	.. matching 14-char length of original band[hex]
		//
		customLOREExportID := fmt.Sprintf("band%s%s",
			strings.ToLower(viper.GetString(cConfigCosmFourCC)),
			strings.ToLower(strings.ReplaceAll(cmdJamToExport, "jam_", "jam")),
		)
		if len(customLOREExportID) != 14 {
			SysLog.Fatal("LORE export ID should be 14 characters long, matching original Endlesss band IDs", zap.String("LoreExID", customLOREExportID))
		}

		SysLog.Info("Exporting Jam",
			zap.String("COSMID", cmdJamToExport),
			zap.String("CouchID", lutID.CouchID),
			zap.String("LoreExID", customLOREExportID),
		)

		// ring ring mr couch
		couchClient, err := connectToCouchDB()
		if err != nil {
			SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
		}
		defer couchClient.Close()

		// connect to the jam's couch database
		jamDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", lutID.CouchID))

		// pull the current Profile doc
		var currentJamProfile JamDatabaseProfileUpdate
		err = jamDb.Get(context.TODO(), "Profile").ScanDoc(&currentJamProfile)
		if err != nil {
			SysLog.Fatal("Unable to fetch jam Profile document", zap.String("COSMID", cmdJamToExport), zap.String("CouchID", lutID.CouchID), zap.Error(err))
		}

		// common yaml/tar base filename
		orxBasePath := fmt.Sprintf("orx.[%s]_%s.%s", strings.ToLower(cmdServerNamePrefix), sanitiseNameForPath(currentJamProfile.DisplayName, '_', false), customLOREExportID)

		// we will write the yaml line by line, open it upfront
		yamlFile, err := os.Create(fmt.Sprintf("%s.yaml", orxBasePath))
		if err != nil {
			SysLog.Fatal("Unable to create output YAML", zap.Error(err))
		}
		defer yamlFile.Close()

		// write the standard header describing the export
		yamlFile.WriteString(fmt.Sprintf("# export from OUROCOSM private server '%s'\n", cmdServerNamePrefix))
		yamlFile.WriteString(fmt.Sprintf("export_time_unix: %d\n", time.Now().Unix()))
		yamlFile.WriteString("export_ouroveon_version: \"1.1.4\"\n") // compliant with 1.1.4, so we pretend
		yamlFile.WriteString(fmt.Sprintf("jam_name: \"[%s] %s\"\n", cmdServerNamePrefix, currentJamProfile.DisplayName))
		yamlFile.WriteString(fmt.Sprintf("jam_couch_id: \"%s\"\n", customLOREExportID))

		{
			// walk the riffs
			resultSet := jamDb.Query(context.TODO(), "types", "rifffsByCreateTime", kivik.Params(map[string]interface{}{
				"descending":   false,
				"include_docs": true,
			}))
			defer resultSet.Close()

			SysLog.Info("Riffs ...")
			var riffCount uint32 = 0

			yamlFile.WriteString("# riffs schema\n")
			yamlFile.WriteString("# couch ID, user, creation unix time, root index, root name, scale index, scale name, BPS (float), BPS (hex float), BPM (float), BPM (hex float), bar length, app version, 8x [ stem couch ID, stem gain (float), stem gain (hex float), stem enabled ]\n")
			yamlFile.WriteString("riffs:\n")

			// page through the whole set, emit data to match archival schema
			for resultSet.Next() {
				var resultData JamRiffData
				if err := resultSet.ScanDoc(&resultData); err != nil {
					SysLog.Fatal("Failed while reading riff documents", zap.Error(err))
				}
				if resultSet.Err() != nil {
					SysLog.Fatal("Failed during iteration of riff document set", zap.Error(resultSet.Err()))
				}

				yamlFile.WriteString(fmt.Sprintf(` "%s": [`, resultData.ID))
				yamlFile.WriteString(fmt.Sprintf(`"%s", %d, %d, "%s", %d, "%s", %f, "%s", %f, "%s", %d, %d, `,
					resultData.UserName,
					resultData.Created/1000, // convert from unixmilli
					resultData.Root,
					getRootName(resultData.Root),
					resultData.Scale,
					getScaleName(resultData.Scale),
					resultData.State.Bps,
					strconv.FormatFloat(resultData.State.Bps, 'x', -1, 64),
					bpsToRoundedBPM(resultData.State.Bps),
					strconv.FormatFloat(bpsToRoundedBPM(resultData.State.Bps), 'x', -1, 64),
					resultData.State.BarLength,
					resultData.AppVersion,
				))

				for i := 0; i < 8; i++ {
					stemData := &resultData.State.Playback[i].Slot.Current

					// only write stem CID if its "on" (matching LORE's export)
					stemIDToWrite := stemData.CurrentLoop
					if !stemData.On {
						stemIDToWrite = ""
					}
					yamlFile.WriteString(fmt.Sprintf(`[ "%s", %f, "%s", %s ], `,
						stemIDToWrite,
						stemData.Gain,
						strconv.FormatFloat(stemData.Gain, 'x', -1, 64),
						strconv.FormatBool(stemData.On),
					))
				}

				yamlFile.WriteString(fmt.Sprintf(` %f ]`, resultData.Magnitude))
				yamlFile.WriteString("\n")

				riffCount++
			}
			SysLog.Info(fmt.Sprintf(" ... wrote %d riffs", riffCount))
		}
		{
			// walk the stems
			resultSet := jamDb.Query(context.TODO(), "types", "loopsByCreateTime", kivik.Params(map[string]interface{}{
				"descending":   false,
				"include_docs": true,
			}))
			defer resultSet.Close()

			SysLog.Info("Stems ...")
			var stemCount uint32 = 0
			var stemDownloads uint32 = 0
			stemFilePaths := []string{}

			yamlFile.WriteString("# stems schema\n")
			yamlFile.WriteString("# couch ID, file endpoint, file bucket, file key, file MIME, file length in bytes, sample rate, creation unix time, preset, user, colour hex, BPS (float), BPS (hex float), BPM (float), BPM (hex float), length 16ths, original pitch, bar length, is-drum, is-note, is-bass, is-mic\n")
			yamlFile.WriteString("stems:\n")

			// same as before, just stems now
			for resultSet.Next() {
				var resultData JamStemData
				if err := resultSet.ScanDoc(&resultData); err != nil {
					SysLog.Fatal("Failed while reading riff documents", zap.Error(err))
				}
				if resultSet.Err() != nil {
					SysLog.Fatal("Failed during iteration of riff document set", zap.Error(resultSet.Err()))
				}

				yamlFile.WriteString(fmt.Sprintf(` "%s": [`, resultData.ID))
				yamlFile.WriteString(fmt.Sprintf(`"%s", "%s", "%s", "%s", %d, %d, %d, "%s", "%s", "%s", %f, "%s", %f, "%s", %d, %d, %d, %s, %s, %s, %s ]`,
					resultData.CdnAttachments.OggAudio.Endpoint,
					"",
					resultData.CdnAttachments.OggAudio.Key,
					resultData.CdnAttachments.OggAudio.Mime,
					resultData.CdnAttachments.OggAudio.Length,
					int32(resultData.SampleRate),
					resultData.Created/1000, // convert from unixmilli
					resultData.PresetName,
					resultData.CreatorUserName,
					resultData.PrimaryColour,
					resultData.Bps,
					strconv.FormatFloat(resultData.Bps, 'x', -1, 64),
					bpsToRoundedBPM(resultData.Bps),
					strconv.FormatFloat(bpsToRoundedBPM(resultData.Bps), 'x', -1, 64),
					resultData.Length16Ths,
					int32(resultData.OriginalPitch),
					resultData.BarLength,
					strconv.FormatBool(resultData.IsDrum),
					strconv.FormatBool(resultData.IsNote),
					strconv.FormatBool(resultData.IsBass),
					strconv.FormatBool(resultData.IsMic),
				))
				yamlFile.WriteString("\n")

				stemCount++

				// stem download server was specified
				if len(cmdStemS3Server) > 0 {

					// create a LORE-cache compatible output path
					stemPath := filepath.Join("_stems", customLOREExportID, resultData.ID[0:1])
					os.MkdirAll(stemPath, os.ModePerm)

					// create from/to locations
					stemDownloadUrl := fmt.Sprintf("https://%s/%s", cmdStemS3Server, resultData.CdnAttachments.OggAudio.Key)
					stemDownloadFile := filepath.Join(stemPath, resultData.ID)

					stemFilePaths = append(stemFilePaths, stemDownloadFile)

					// download if we don't already have it
					if _, err := os.Stat(stemDownloadFile); errors.Is(err, os.ErrNotExist) {
						err = downloadStem(httpClient, resultData.CdnAttachments.OggAudio.Length, stemDownloadFile, stemDownloadUrl)
						if err != nil {
							SysLog.Fatal("Stem download failed", zap.Error(err))
							os.Remove(stemDownloadFile)
						}
						stemDownloads++
					}
				}
			}
			SysLog.Info(fmt.Sprintf(" ... wrote %d stems", stemCount))
			if stemDownloads > 0 {
				SysLog.Info(fmt.Sprintf(" ... downloaded %d stems", stemDownloads))
			}
			// if we were processing downloaded stems, emit the collected list of stem files into the final LORE-importable .TAR
			if len(stemFilePaths) > 0 {

				// to pacify LORE, we also write out all the required directory structure - do this first to get the structure built upfront
				directoryBasePaths := []string{}
				err := filepath.Walk(filepath.Join("_stems", customLOREExportID), func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if info.IsDir() {
						directoryBasePaths = append(directoryBasePaths, path)
					}
					return nil
				})
				if err != nil {
					SysLog.Fatal("TAR preparation failed", zap.Error(err))
				}

				// bolt on all the stems to write to the TAR
				finalTARLayout := append(directoryBasePaths, stemFilePaths[:]...)

				// remove (if required) and recreate the archive
				tarOutputFile := fmt.Sprintf("%s.tar", orxBasePath)
				os.Remove(tarOutputFile)
				err = createTarArchive(finalTARLayout, "_stems/", tarOutputFile)
				if err != nil {
					SysLog.Fatal("TAR archive creation failed", zap.Error(err))
					os.Remove(tarOutputFile)
				}
				SysLog.Info(" ... TAR archive written")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)

	exportCmd.Flags().StringVarP(&cmdJamToExport, "jam", "j", "", "(required) COSMID jam ID to export")
	exportCmd.MarkFlagRequired("jam")
	exportCmd.Flags().StringVarP(&cmdServerNamePrefix, "prefix", "p", "", "(required) Server name prefix applied to each jam export")
	exportCmd.MarkFlagRequired("prefix")
	exportCmd.Flags().StringVarP(&cmdStemS3Server, "stem", "s", "", "if given, talk to this S3 server to fetch the stems and bake them into a .tar")
}
