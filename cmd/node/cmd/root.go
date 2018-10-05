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
	"log"
	"os"
	"os/signal"
	"syscall"

	kcoinlog "github.com/kowala-tech/kcoin/client/log"
	"github.com/kowala-tech/p2p-poc/core"
	"github.com/kowala-tech/p2p-poc/node"
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
	Use:   "node",
	Short: "Network node",
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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.node.yaml)")
	rootCmd.PersistentFlags().StringVar(&listenAddr, "addr", "/ip4/127.0.0.1/tcp/33445", "listen address")
	rootCmd.PersistentFlags().IntVar(&verbosity, "verbosity", int(kcoinlog.LvlInfo), "log verbosity (0-9)")
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

		// Search config in home directory with name ".node" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".node")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func run(cmd *cobra.Command, args []string) error {
	node := makeFullNode()
	startNode(node)
	node.Wait()
	return nil
}

func makeFullNode() *node.Node {
	node, cfg := makeConfigNode()
	registerCoreService(node, cfg.Core)
	return node
}

func registerCoreService(stack *node.Node, cfg core.Config) {
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, err := core.New(ctx, cfg)
		return fullNode, err
	}); err != nil {
		log.Fatal("Failed to register the core service")
	}
}

type GlobalConfig struct {
	Core core.Config
	Node node.Config
}

func makeConfigNode() (*node.Node, GlobalConfig) {
	cfg := GlobalConfig{
		Node: node.DefaultConfig,
		Core: core.DefaultConfig,
	}

	node := node.New(context.Background(), cfg.Node)

	return node, cfg
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// validator.
func startNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		kcoinlog.Crit("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		kcoinlog.Info("Got interrupt, shutting down...")
		go stack.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				kcoinlog.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		//debug.Exit() // ensure trace and CPU profile data is flushed.
		//debug.LoudPanic("boom")
	}()
}
