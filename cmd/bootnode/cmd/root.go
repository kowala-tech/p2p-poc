// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	dstore "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/kowala-tech/kcoin/client/log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	verbosity  int
	vmodule    string
	listenAddr string
)

var rootCmd = &cobra.Command{
	Use:   "bootnode",
	Short: "Network bootnode",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: run,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bootnode.yaml)")
	rootCmd.PersistentFlags().StringVar(&listenAddr, "addr", "/ip4/127.0.0.1/tcp/33445", "listen address")
	rootCmd.PersistentFlags().IntVar(&verbosity, "verbosity", int(log.LvlInfo), "log verbosity (0-9)")
	rootCmd.PersistentFlags().StringVar(&vmodule, "vmodule", "", "log verbosity pattern")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".bootnode" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".bootnode")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func run(cmd *cobra.Command, args []string) error {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stdout, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(verbosity))
	glogger.Vmodule(vmodule)
	log.Root().SetHandler(glogger)

	log.Info("Initialising bootnode")

	ctx := context.Background()

	privKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.New(rand.NewSource(0)))
	if err != nil {
		return err
	}

	host, err := libp2p.New(ctx, libp2p.Identity(privKey), libp2p.ListenAddrStrings(listenAddr))
	if err != nil {
		return err
	}

	dhtClient := dht.NewDHT(ctx, host, sync.MutexWrap(dstore.NewMapDatastore()))
	if err := dhtClient.Bootstrap(ctx); err != nil {
		return err
	}

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGTERM)

	select {
	case <-signals:
	}

	return nil
}
