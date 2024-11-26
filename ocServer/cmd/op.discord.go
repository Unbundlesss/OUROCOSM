//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var cmdDiscordBotToken = ""
var cmdDiscordApp = ""
var cmdDiscordGuild = ""

var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "anyonejamming",
		Description: "Check the OUROCOSM server and report recent activity",
	},
}

func fetchServerStatus() (*StatusResponse, error) {

	cosmStatus := fmt.Sprintf("http://%s:%s/cosm/v1/status", viper.GetString(cConfigCosmInternalHost), viper.GetString(cConfigCosmInternalPort))

	resp, err := http.Get(cosmStatus)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make GET request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var status StatusResponse
	err = json.Unmarshal(body, &status)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal JSON")
	}
	return &status, nil
}

func handleJamQueryCmd(s *discordgo.Session, i *discordgo.InteractionCreate) {

	builder := new(strings.Builder)

	serverStatus, err := fetchServerStatus()
	if err != nil {
		SysLog.Warn("Failed to talk to server", zap.Error(err))
		builder.WriteString("Failed to talk to the server, sorry.")
	} else {
		builder.WriteString("Last riff in a public jam was ")
		builder.WriteString(serverStatus.MostRecentPublicJamChangeText)
		builder.WriteString(" by `")
		builder.WriteString(serverStatus.MostRecentPublicJamUser)
		builder.WriteString("` in **")
		builder.WriteString(serverStatus.MostRecentPublicJamName)
		builder.WriteString("**.")
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		SysLog.Warn("InteractionRespond failed", zap.Error(err))
	}
}

var discordCmd = &cobra.Command{
	Use:   "discord",
	Short: "Run a utility Discord bot that can talk to an OUROCOSM server",
	Long:  `Run a utility Discord bot that can talk to an OUROCOSM server`,
	Run: func(cmd *cobra.Command, args []string) {

		discord, err := discordgo.New("Bot " + cmdDiscordBotToken)
		if err != nil {
			SysLog.Fatal("Unable to create Discord session", zap.Error(err))
		}

		discord.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if i.Type != discordgo.InteractionApplicationCommand {
				return
			}

			data := i.ApplicationCommandData()
			if data.Name != "anyonejamming" {
				return
			}

			handleJamQueryCmd(s, i)
		})

		discord.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
			SysLog.Info("Discord login", zap.String("User", r.User.String()))
		})

		_, err = discord.ApplicationCommandBulkOverwrite(cmdDiscordApp, cmdDiscordGuild, commands)
		if err != nil {
			SysLog.Fatal("ApplicationCommandBulkOverwrite() failed", zap.Error(err))
		}

		err = discord.Open()
		if err != nil {
			SysLog.Fatal("Unable to connect to Discord", zap.Error(err))
		}

		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, os.Interrupt)
		<-sigch

		err = discord.Close()
		if err != nil {
			SysLog.Fatal("Unable to close connection to Discord gracefully", zap.Error(err))
		}
		SysLog.Info("Discord connection closed")
	},
}

func init() {
	rootCmd.AddCommand(discordCmd)

	discordCmd.Flags().StringVarP(&cmdDiscordBotToken, "token", "t", "", "Bot auth token")
	discordCmd.MarkFlagRequired("token")
	discordCmd.Flags().StringVarP(&cmdDiscordApp, "app", "a", "", "App ID")
	discordCmd.MarkFlagRequired("app")
	discordCmd.Flags().StringVarP(&cmdDiscordGuild, "guild", "g", "", "Guild ID")
	discordCmd.MarkFlagRequired("guild")
}
