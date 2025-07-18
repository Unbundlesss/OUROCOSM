package cmd

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	kivik "github.com/go-kivik/kivik/v4"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func deduceOutputParametersForJam(jamToExport string) (string, string, string) {

	// if we're exporting a public/private jam, it begins with the COSMID prefix "jam_"
	// otherwise, we're (presumably) asking for a personal user's own jam, handled after this block
	if strings.HasPrefix(jamToExport, "jam_") {

		// lookup couch ID for access
		idBank := SysBankIDs.Bank()
		lutID, ok := idBank.Entries[jamToExport]
		if !ok {
			SysLog.Fatal("Unable to resolve COSMID to Endlesss jam IDs", zap.String("COSMID", jamToExport))
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
			strings.ToLower(strings.ReplaceAll(jamToExport, "jam_", "jam")),
		)
		if len(customLOREExportID) != 14 {
			SysLog.Fatal("LORE export ID should be 14 characters long, matching original Endlesss band IDs", zap.String("LoreExID", customLOREExportID))
		}

		SysLog.Info("Exporting Jam",
			zap.String("COSMID", jamToExport),
			zap.String("CouchID", lutID.CouchID),
			zap.String("LoreExID", customLOREExportID),
		)

		return lutID.CouchID, customLOREExportID, ""
	} else {

		// personal jams need the server prefix too, to differentiate them from the OG ones
		customLOREPersonalID := fmt.Sprintf("%s_%s",
			strings.ToLower(viper.GetString(cConfigCosmFourCC)),
			strings.ToLower(jamToExport),
		)

		SysLog.Info("Exporting Personal Jam",
			zap.String("CouchID", jamToExport),
			zap.String("LoreExID", customLOREPersonalID),
		)

		return jamToExport, customLOREPersonalID, jamToExport
	}
}

func exportJamToDisk(outputDir string, jamToExport string, serverNamePrefix string, stemS3Server string, ignoreMissingStems bool) ([]string, error) {

	if len(jamToExport) == 0 {
		return nil, fmt.Errorf("Jam to export cannot be null")
	}

	resultingFiles := []string{}

	// when yanking stems from our server, we might be pointing at localhost or similar which
	// (as it has to be over https) would kick up cert issues, so skip verification by default
	httpTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: httpTransport}

	exportCouchID, exportLOREID, jamProfileDisplayNameUnsanitised := deduceOutputParametersForJam(jamToExport)

	// ring ring mr couch
	couchClient, err := connectToCouchDB()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("Connection to CouchDB failed"), err)
	}
	defer couchClient.Close()

	// connect to the jam's couch database
	jamDb := couchClient.DB(fmt.Sprintf("user_appdata$%s", exportCouchID))

	// no provided display name from deduceOutputParametersForJam(), get it from the jam's Profile doc
	if len(jamProfileDisplayNameUnsanitised) == 0 {
		// pull the current Profile doc
		var currentJamProfile JamDatabaseProfileUpdate
		err = jamDb.Get(context.TODO(), "Profile").ScanDoc(&currentJamProfile)
		if err != nil {
			return nil, errors.Join(fmt.Errorf("Unable to fetch jam Profile document COSMID:[%s] CouchID:[%s]", jamToExport, exportCouchID), err)
		}
		jamProfileDisplayNameUnsanitised = currentJamProfile.DisplayName
	}
	jamProfileDisplayName := sanitiseNameForPath(jamProfileDisplayNameUnsanitised, '_', false)

	// common yaml/tar base filename
	orxBasePath := fmt.Sprintf("orx.[%s]_%s.%s", strings.ToLower(serverNamePrefix), jamProfileDisplayName, exportLOREID)

	SysLog.Info("Jam Profile",
		zap.String("Name", jamProfileDisplayName),
		zap.String("Output", orxBasePath),
	)

	// we will write the yaml line by line, open it upfront
	yamlFileRoot := path.Join(outputDir, "_archives")
	os.MkdirAll(yamlFileRoot, os.ModePerm)

	yamlFilePath := path.Join(yamlFileRoot, fmt.Sprintf("%s.yaml", orxBasePath))
	yamlFile, err := os.Create(yamlFilePath)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("Unable to create output YAML"), err)
	}
	defer yamlFile.Close()
	resultingFiles = append(resultingFiles, yamlFilePath)

	// write the standard header describing the export
	yamlFile.WriteString(fmt.Sprintf("# export from OUROCOSM private server '%s'\n", serverNamePrefix))
	yamlFile.WriteString(fmt.Sprintf("export_time_unix: %d\n", time.Now().Unix()))
	yamlFile.WriteString("export_ouroveon_version: \"1.1.4\"\n") // compliant with 1.1.4, so we pretend
	yamlFile.WriteString(fmt.Sprintf("jam_name: \"[%s] %s\"\n", serverNamePrefix, jamProfileDisplayNameUnsanitised))
	yamlFile.WriteString(fmt.Sprintf("jam_couch_id: \"%s\"\n", exportLOREID))

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
				return nil, errors.Join(fmt.Errorf("Failed while reading riff documents"), err)
			}
			if resultSet.Err() != nil {
				return nil, errors.Join(fmt.Errorf("Failed during iteration of riff document set"), resultSet.Err())
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
				return nil, errors.Join(fmt.Errorf("Failed while reading stem documents"), err)
			}
			if resultSet.Err() != nil {
				return nil, errors.Join(fmt.Errorf("Failed during iteration of stem document set"), resultSet.Err())
			}

			cdnEndpoint := getActiveEndpoint(resultData)

			yamlFile.WriteString(fmt.Sprintf(` "%s": [`, resultData.ID))
			yamlFile.WriteString(fmt.Sprintf(`"%s", "%s", "%s", "%s", %d, %d, %d, "%s", "%s", "%s", %f, "%s", %f, "%s", %d, %d, %d, %s, %s, %s, %s ]`,
				cdnEndpoint.Endpoint,
				"",
				cdnEndpoint.Key,
				cdnEndpoint.Mime,
				cdnEndpoint.Length,
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
			if len(stemS3Server) > 0 {

				// create a LORE-cache compatible output path
				stemPath := filepath.Join(outputDir, "_stems", exportLOREID, resultData.ID[0:1])
				os.MkdirAll(stemPath, os.ModePerm)

				cdnEndpoint := getActiveEndpoint(resultData)

				// create from/to locations
				stemDownloadUrl := fmt.Sprintf("https://%s/%s", stemS3Server, cdnEndpoint.Key)
				stemDownloadFile := filepath.Join(stemPath, resultData.ID)

				stemFileExist := false

				// download if we don't already have it
				if _, err := os.Stat(stemDownloadFile); errors.Is(err, os.ErrNotExist) {
					err = downloadStem(httpClient, cdnEndpoint.Length, stemDownloadFile, stemDownloadUrl)
					if err != nil {
						if !ignoreMissingStems {
							return nil, errors.Join(fmt.Errorf("Stem download failed [%s]", stemDownloadUrl), err)
						} else {
							SysLog.Warn("Stem download failed", zap.Error(err), zap.String("url", stemDownloadUrl))
						}
						os.Remove(stemDownloadFile)
					} else {
						// file downloaded ok
						stemDownloads++
						stemFileExist = true
					}
				} else {
					// file already in cache
					stemFileExist = true
				}

				if stemFileExist {
					stemFilePaths = append(stemFilePaths, stemDownloadFile)
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
			err := filepath.Walk(filepath.Join(outputDir, "_stems", exportLOREID), func(path string, info os.FileInfo, err error) error {
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

			SysLog.Info(fmt.Sprintf("tar creation with %d entries", len(finalTARLayout)))

			// remove (if required) and recreate the archive
			tarOutputFile := path.Join(outputDir, "_archives", fmt.Sprintf("%s.tar", orxBasePath))
			os.Remove(tarOutputFile)
			err = createTarArchive(finalTARLayout, path.Join(outputDir, "_stems"), tarOutputFile)
			if err != nil {
				SysLog.Fatal("TAR archive creation failed", zap.Error(err))
				os.Remove(tarOutputFile)
			}
			SysLog.Info(" ... TAR archive written")

			resultingFiles = append(resultingFiles, tarOutputFile)
		}

		return resultingFiles, nil
	}
}
