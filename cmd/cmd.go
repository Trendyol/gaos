/*
Copyright 2020 The Gaos Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"runtime"
	"strings"
	"zeus-gitlab.trendyol.com/general/gaos/executor"
	"zeus-gitlab.trendyol.com/general/gaos/logger"
	"zeus-gitlab.trendyol.com/general/gaos/runner"
)

func Execute(version, builtBy, date, commit string) {
	info := fmt.Sprintf(
		"Gaos %s (%s, %s, %s) on %s (%s)",
		version,
		builtBy,
		date,
		commit,
		runtime.GOOS,
		runtime.GOARCH,
	)

	var config executor.Config
	var scenario, execute string

	var cmd = &cobra.Command{
		Use: "gaos",
	}

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run Gaos server on localhost",
		Long:  "Run Gaos server on your localhost",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {

			gaos, err := runner.New(scenario)

			if err != nil {
				logger.Error(err)
				os.Exit(1)
				return
			}

			if len(execute) > 0 {
				gaos.Run(strings.Split(execute, ",")...)
			} else {
				gaos.Run()
			}
		},
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Args:  cobra.NoArgs,
		Short: "Start Gaos server on given engine (Docker, K8S)",
		Long:  "Start Gaos server on given engine (Docker, K8S)",
		Run: func(cmd *cobra.Command, args []string) {

			ex, err := executor.NewExecutor(config)

			if err != nil {
				logger.Error(err)
				os.Exit(1)
				return
			}

			err = ex.Run()

			if err != nil {
				logger.Error(err)
				os.Exit(1)
				return
			}
		},
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Gaos",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(info)
		},
	}

	//run flags
	runCmd.Flags().StringVarP(&execute, "execute", "x", "", "execute scenario services")
	runCmd.Flags().StringVarP(&scenario, "scenario", "s", "./scenario.json", "scenario file input")

	//start flags
	startCmd.Flags().StringVarP(&config.Environment, "environment", "e", "local", "gaos running environment {docker, k8s}")
	startCmd.Flags().StringVarP(&config.Scenario, "scenario", "s", "./scenario.json", "scenario file input")
	startCmd.Flags().IntVarP(&config.ContinueOnFailure, "--continue-on-failure", "y", 0, "continue on failure {0: ask prompt, 1: continue on failure, 2: break on failure}. default: 0")

	//docker environment flags
	startCmd.Flags().StringVarP(&config.Timeout, "timeout", "t", "5m", "client timeout")
	startCmd.Flags().StringVarP(&config.Registry, "registry", "r", "", "image registry")
	startCmd.Flags().StringVarP(&config.Username, "username", "u", "", "image registry username")
	startCmd.Flags().StringVarP(&config.Password, "password", "p", "", "image registry password")

	//kubernetes environment flags
	startCmd.Flags().StringVarP(&config.Config, "config", "c", "minikube", "choose k8s config")
	startCmd.Flags().StringVarP(&config.Namespace, "namespace", "n", "default", "choose namespace")
	startCmd.Flags().StringVarP(&config.Cpu, "cpu", "", "500m", "cpu limit")
	startCmd.Flags().StringVarP(&config.Memory, "memory", "", "500Mi", "memory limit")
	startCmd.Flags().StringVarP(&config.Secret, "secret", "", "", "secret key name")
	startCmd.Flags().StringVarP(&config.Replica, "replica", "", "1", "replica count")

	cmd.AddCommand(runCmd, startCmd, versionCmd)

	cmd.SetVersionTemplate(info)

	_ = cmd.Execute()
}

