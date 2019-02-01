package main

import (
	"fmt"
	"github.com/op/go-logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gitlab.chainnova.com/motcert-backend/app/fabricClient"
	"os"
	"runtime"
	"strings"
)

const (
	cmdRoot = "appserver"
)

var (
	// Logging
	logger            = logging.MustGetLogger("Motcert.AppServer")
	versionFlag       bool
	FabricSetupEntity *fabricClient.FabricSetup

	appStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Starts the app.",
		Long:  `Starts a app that interacts with the network.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			setFabricSdk(false)
			return appService(args)
		},
	}
	updateStartCmd = &cobra.Command{
		Use:   "update",
		Short: "update the app",
		Long:  "update a app that interacts with the network",
		RunE: func(cmd *cobra.Command, args []string) error {
			setFabricSdk(true)
			return appService(args)
		},
	}
)

// The main command describes the service and
// defaults to printing the help message.
var mainCmd = &cobra.Command{
	Use: "app",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if versionFlag {
			VersionPrint()
		} else {
			cmd.HelpFunc()(cmd, args)
		}
	},
}

func main() {
	// Logging
	var formatter = logging.MustStringFormatter(
		`%{color}[%{module}] %{shortfunc} [%{shortfile}] -> %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
	logging.SetFormatter(formatter)

	// viper init
	viper.AddConfigPath("../")
	viper.SetConfigName(cmdRoot)
	viper.SetEnvPrefix(cmdRoot)
	viper.AutomaticEnv()
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	// Define command-line flags that are valid
	mainFlags := mainCmd.PersistentFlags()
	mainFlags.BoolVarP(&versionFlag, "version", "v", false, "Display current version")
	mainCmd.AddCommand(VersionCmd())
	mainCmd.AddCommand(appStartCmd)
	mainCmd.AddCommand(updateStartCmd)
	mainCmd.AddCommand(testCmd())
	runtime.GOMAXPROCS(viper.GetInt("app.gomaxprocs"))

	// Close SDK
	defer FabricSetupEntity.CloseSDK()

	if mainCmd.Execute() != nil {
		os.Exit(1)
	}
	logger.Info("Exiting.....")
}

func setFabricSdk(hasChannel bool) {
	// Definition of the Fabric SDK properties
	FabricSetupEntity = &fabricClient.FabricSetup{
		// Network parameters
		OrdererID: "orderer.cert.mot.gov.cn",

		// Channel parameters
		ChannelID:     "motcert",
		ChannelConfig: os.Getenv("GOPATH") + "/src/gitlab.chainnova.com/motcert-backend/fixtures/artifacts/motcert.channel.tx",

		// Chaincode parameters
		ChainCodeID:     "motcert-cc1",
		ChaincodeGoPath: os.Getenv("GOPATH"),
		ChaincodePath:   "gitlab.chainnova.com/motcert-backend/app/chaincode/",
		OrgAdmin:        "Admin",
		OrgName:         "org1",
		ConfigFile:      "../config.yaml",

		// User parameters
		UserName: "User1",
	}

	// Initialization of the Fabric SDK from the previously set properties
	err := FabricSetupEntity.Initialize(hasChannel)
	if err != nil {
		logger.Errorf("Unable to initialize the Fabric SDK: %v\n", err)
		return
	}

	//Install and instantiate the chaincode
	err = FabricSetupEntity.InstallAndInstantiateCC(hasChannel)
	if err != nil {
		logger.Errorf("Unable to install and instantiate the chaincode: %v\n", err)
		return
	}
}
