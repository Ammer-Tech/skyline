package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var debug bool
var configLocation string

var rootCmd = &cobra.Command{
	Use:   "skyline",
	Short: "Secure SMTP on the horizon",
	Long: `Skyline bridges the need for a classic SMTP server and the security measures found
in modern cloud providers.`,
	PersistentPreRun: toggleDebug,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func toggleDebug(cmd *cobra.Command, args []string) {
	logLevel := &slog.LevelVar{}

	if debug {
		logLevel.Set(slog.LevelDebug)
		slog.Debug("Debug logs enabled")
	} else {
		logLevel.Set(slog.LevelInfo)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "verbose logging")
	rootCmd.PersistentFlags().StringVarP(&configLocation, "config", "c", "skyline.yaml", "specify config file location")
	cobra.OnInitialize(initConfig)

	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		slog.Error("could not bind root persistent flags to viper config", "error", err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	slog.Info("Current config", viper.GetString("config"))
	if viper.Get("config") == "skyline.yaml" {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.AddConfigPath("/opt/skyline")
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".skyline")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		viper.SetEnvPrefix("SL")
		viper.AutomaticEnv() // read in environment variables that match
		slog.Info("Looking for config", slog.String("directory", home))
		slog.Info("Looking for config", slog.String("directory", "/opt/skyline"))
	}
	if viper.GetString("config") != "skyline.yaml" {
		viper.SetConfigFile(configLocation)
	}

	// If a config file is found, read it in.

	if err := viper.ReadInConfig(); err == nil {
		slog.Info("Using config file:", slog.String("file", viper.ConfigFileUsed()))
	}
}
