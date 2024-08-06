//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"archive/tar"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"
)

// matching lookup from LORE / Endlesss
func getRootName(n int) string {
	return [...]string{
		/*  0 */ "C",
		/*  1 */ "Db", // c#
		/*  2 */ "D",
		/*  3 */ "Eb", // d#
		/*  4 */ "E",
		/*  5 */ "F",
		/*  6 */ "F#", // g flat
		/*  7 */ "G",
		/*  8 */ "Ab", // g sharp
		/*  9 */ "A",
		/* 10 */ "Bb", // a#
		/* 11 */ "B",
	}[n]
}

// matching lookup from LORE / Endlesss
func getScaleName(n int) string {
	return [...]string{
		"major",
		"dorian",
		"phrygian",
		"lydian",
		"mixoly",
		"minor",
		"locrian",
		"minor_pent",
		"major_pent",
		"susp_pent",
		"blues_mnr_p",
		"blues_mjr_p",
		"harmonic_mnr",
		"melodic_mnr",
		"dbl_harmonic",
		"blues",
		"whole",
		"chromatic",
	}[n]
}

func bpsToRoundedBPM(bps float64) float64 {
	return (math.Ceil((bps*60.0)*100.0) / 100.0)
}

// ported version of what we do in LORE; recreate the same pathname sanitiser output
func sanitiseNameForPath(source string, replacementChar rune, allowWhitespace bool) string {
	var dest strings.Builder
	dest.Grow(len(source))

	endsWithWhitespace := false

	for i := 0; i < len(source); {
		cp, size := utf8.DecodeRuneInString(source[i:])
		i += size

		// blitz control characters
		if (cp >= 0x00 && cp <= 0x1f) || (cp >= 0x80 && cp <= 0x9f) {
			cp = replacementChar
		}

		// strip out problematic pathname characters
		switch cp {
		case '/', '?', '<', '>', '\\', ':', '*', '|', '"', '~', '.':
			cp = replacementChar
		}

		switch cp {
		case ' ', '\t', '\n':
			if !allowWhitespace {
				cp = replacementChar
			}
			endsWithWhitespace = true
		default:
			endsWithWhitespace = false
		}

		dest.WriteRune(cp)
	}

	if endsWithWhitespace {
		dest.WriteRune('_')
	}

	return strings.ToLower(dest.String())
}

// format of endlesss.publics.json that ships with LORE; we only used this temporarily to
// generate the embedded IDBank data from LORE's public jam JSON
type LOREPublic struct {
	Jams []struct {
		BandID                  string `json:"band_id"`
		InviteID                string `json:"invite_id"`
		ListenID                string `json:"listen_id"`
		JamName                 string `json:"jam_name"`
		EarliestUser            string `json:"earliest_user"`
		LatestUser              string `json:"latest_user"`
		EarliestUnixtime        int    `json:"earliest_unixtime"`
		LatestUnixtime          int    `json:"latest_unixtime"`
		EstimatedDaysOfActivity int    `json:"estimated_days_of_activity"`
		TotalRiffs              int    `json:"total_riffs"`
		SubscribedMemberCount   int    `json:"subscribed_member_count"`
	} `json:"jams"`
}

// fetch a stem from somewhere, failing if we don't get the size the database told us to expect, write to given filepath
func downloadStem(httpClient *http.Client, expectedLength int, filepath string, url string) error {

	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error while downloading: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	bytesWritten, err := io.Copy(out, resp.Body)
	if int64(expectedLength) != bytesWritten {
		return fmt.Errorf("stem file size mismatch, got %d, expected %d", bytesWritten, expectedLength)
	}
	return err
}

// based on https://www.arthurkoziel.com/writing-tar-gz-files-in-go/
func createTarArchive(files []string, baseRelativePath, outputFile string) error {

	buf, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer buf.Close()

	tw := tar.NewWriter(buf)
	defer tw.Close()

	// Iterate over files and add them to the tar archive
	for _, file := range files {
		err := addToArchive(tw, baseRelativePath, file)
		if err != nil {
			return err
		}
	}

	return nil
}
func addToArchive(tw *tar.Writer, baseRelativePath, filename string) error {
	// Open the file which will be written into the archive
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get FileInfo about our file providing file size, mode, etc.
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create a tar Header from the FileInfo data
	header, err := tar.FileInfoHeader(info, info.Name())
	if err != nil {
		return err
	}

	// Use full path as name (FileInfoHeader only takes the basename)
	// If we don't do this the directory strucuture would
	// not be preserved
	// https://golang.org/src/archive/tar/common.go?#L626
	header.Name = strings.Replace(strings.Replace(filename, "\\", "/", -1), baseRelativePath, "", -1)

	// Write file header to the tar archive
	err = tw.WriteHeader(header)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		// Copy file content to tar archive
		_, err = io.Copy(tw, file)
		if err != nil {
			return err
		}
	}

	return nil
}
