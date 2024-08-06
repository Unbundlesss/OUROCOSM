//
// OUROCOSM // private Endlesss servers proof-of-concept // ishani.org 2024 // GPLv3
// https://github.com/Unbundlesss/OUROCOSM
//

package cmd

import (
	"os"

	"github.com/Unbundlesss/OUROCOSM/ocServer/cmd/internal/util"
	"github.com/mattn/go-colorable"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var SysLog *zap.Logger
var SysBankIDs *util.JamIDs

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "ocServer",
	Short: "OUROCOSM SERVER",
	Long:  `Tools to maintain and operate an OUROCOSM instance\nishani.org 2024`,
}

func Execute() {

	defer SysLog.Sync()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// build app-wide logger instance
	logEncoderConfig := zap.NewDevelopmentEncoderConfig()
	logEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	SysLog = zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(logEncoderConfig),
		zapcore.AddSync(colorable.NewColorableStdout()),
		zapcore.DebugLevel,
	))

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./ourocosm.server.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {

	var err error

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("ourocosm.server")
	}

	viper.SetEnvPrefix("COSM")
	viper.AutomaticEnv()

	if err = viper.ReadInConfig(); err == nil {
		SysLog.Info("Loaded configuration", zap.String("file", viper.ConfigFileUsed()))
	} else {
		SysLog.Warn("Could not load configuration, do you have the ourocosm.server.yaml file available?")
	}

	SysBankIDs, err = util.LoadJamIDBanks()
	if err != nil {
		SysLog.Fatal("Failed to load embedded jam IDs", zap.Error(err))
	}
	SysLog.Info("Loaded embedded jam IDs", zap.Int("count", len(SysBankIDs.Bank().Entries)))
}
