//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"context"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var cmdNewUserName = ""
var cmdNewUserPass = ""
var cmdNewUserBio = ""

var (
	usernameInvalidCharacterRegExp = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

var newUserCmd = &cobra.Command{
	Use:   "newuser",
	Short: "Add a new user to the server",
	Long:  `Add a new user to the server`,
	Run: func(cmd *cobra.Command, args []string) {

		// stop trying to add empty things or long things
		if len(cmdNewUserName) == 0 || len(cmdNewUserName) > 16 {
			SysLog.Fatal("Username cannot be blank, nor longer than 16 letters", zap.String("User", cmdNewUserName))
		}
		// stop trying to make usernames with emoji in or whatever
		if usernameInvalidCharacterRegExp.Match([]byte(cmdNewUserName)) {
			SysLog.Fatal("Username contains invalid symbols - alphanumeric only and underlines only, please", zap.String("User", cmdNewUserName))
		}
		if len(cmdNewUserPass) == 0 {
			SysLog.Fatal("Login password cannot be blank")
		}

		couchClient, err := connectToCouchDB()
		if err != nil {
			SysLog.Fatal("Connection to CouchDB failed", zap.Error(err))
		}
		defer couchClient.Close()

		newUserId := getCouchRecordIDForUser(cmdNewUserName)

		// add our new pal to the users db
		// note the password is generated and internal to the database permissions - it's what Endlesss will use
		// to talk to the users' own solo jam database. i'm mostly just making up how to hand out those token/pwd combos, this will do for now
		// (it's about as leaky as the Endlesss setup was, you could sniff the couchbase password from an auth request and log into Fauxton there too)
		userDB := couchClient.DB("_users")
		_, err = userDB.Put(context.TODO(), newUserId, map[string]interface{}{
			"name":     cmdNewUserName,
			"type":     "user",
			"roles":    []string{"jammers"},
			"password": generateInternalCouchUserPassword(cmdNewUserName),
			"login":    cmdNewUserPass,
			"bio":      cmdNewUserBio,
		})
		if err != nil {
			SysLog.Fatal("Failed to insert new _users record", zap.String("User", cmdNewUserName), zap.String("_id", newUserId), zap.Error(err))
		}

		// database name requires only lowercase characters (a-z), digits (0-9), underline
		usernameForDatabase := strings.ToLower(cmdNewUserName)

		// build our user a new solo jam <3
		soloDB, err := createNewJamDatabase(couchClient, usernameForDatabase)
		if err != nil {
			SysLog.Fatal("Failed to create user database", zap.String("User", cmdNewUserName), zap.Error(err))
		}

		// snag the security block so we can add the user to the members list
		soloSecurity, err := soloDB.Security(context.TODO())
		if err != nil {
			SysLog.Fatal("Failed to acquire user database security", zap.String("User", cmdNewUserName), zap.Error(err))
		}

		// .. add the username to the members-permissions pile
		soloSecurity.Members.Names = append(soloSecurity.Members.Names, cmdNewUserName)

		// write it back
		err = soloDB.SetSecurity(context.TODO(), soloSecurity)
		if err != nil {
			SysLog.Fatal("Failed to reconfigure user database security", zap.String("User", cmdNewUserName), zap.Error(err))
		}

		SysLog.Info("Successfully added new user", zap.String("User", cmdNewUserName))
	},
}

func init() {
	rootCmd.AddCommand(newUserCmd)

	newUserCmd.Flags().StringVarP(&cmdNewUserName, "name", "n", "", "(required) Endlesss user display name")
	newUserCmd.MarkFlagRequired("name")

	newUserCmd.Flags().StringVarP(&cmdNewUserPass, "pass", "p", "", "(required) Endlesss login password")
	newUserCmd.MarkFlagRequired("pass")

	newUserCmd.Flags().StringVarP(&cmdNewUserBio, "bio", "b", "No bio supplied", "Profile page bio text")
}
