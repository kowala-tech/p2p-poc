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
	"fmt"
	"os"

	"github.com/kowala-tech/p2p-poc/p2p"

	"github.com/kowala-tech/kcoin/client/log"
	"github.com/libp2p/go-libp2p-net"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	homedir "github.com/mitchellh/go-homedir"
	maddr "github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	verbosity  int
	listenAddr string
	seed       int
)

var rootCmd = &cobra.Command{
	Use:   "bootnode",
	Short: "Network bootnode",
	Long:  ``,
	RunE:  run,
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
	rootCmd.PersistentFlags().IntVar(&seed, "seed", 0, "random seed for ID generation")
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
	// filter log messages by level flag
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(log.Lvl(verbosity))
	log.Root().SetHandler(glogger)

	config := p2p.DefaultConfig
	config.IsBootnode = true
	config.IDGenerationSeed = seed

	host, err := p2p.NewHost(config)
	if err != nil {
		log.Crit("Failed to create a bootnode", "err", err)
	}

	log.Info("Bootnode is running...", "id", host.ID().Pretty())

	notifiee := &net.NotifyBundle{
		ListenF:      peerConnected,
		ListenCloseF: peerDisconnected,
	}
	host.Network().Notify(notifiee)

	select {}
}

func peerConnected(network net.Network, addr maddr.Multiaddr) {
	log.Info("Peer connected")
	p, err := pstore.InfoFromP2pAddr(addr)
	if err != nil {
		log.Error("Failed to convert addr to peer ID")
		return
	}
	log.Info("Peer connected", "ID", p.ID)
	/*
		newPeer(s.host, addr)
		s.wg.Add(1)
		defer s.wg.Done()
	*/
}

func peerDisconnected(network net.Network, addr maddr.Multiaddr) {
	log.Info("Peer disconnected")
	// @TODO
}

func Connected(net.Network, net.Conn) {
	log.Info("Connected")
}

func Disconnected(net.Network, net.Conn) {
	log.Info("Disconnected")
}

func OpenedStream(net.Network, net.Stream) {
	log.Info("Opened stream")
}

func ClosedStream(net.Network, net.Stream) {
	log.Info("Closed stream")
}
