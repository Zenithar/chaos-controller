// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2025 Datadog, Inc.

package main

import (
	"github.com/spf13/cobra"

	"github.com/DataDog/chaos-controller/api/v1beta1"
	"github.com/DataDog/chaos-controller/injector"
)

var outOfPodsCmd = &cobra.Command{
	Use:   "out-of-pods",
	Short: "Out of pods subcommands",
	Run:   injectAndWait,
	PreRun: func(cmd *cobra.Command, args []string) {
		forced, _ := cmd.Flags().GetBool("forced")

		// prepare spec
		spec := v1beta1.OutOfPodsSpec{
			Forced: forced,
		}

		// create injector
		for _, config := range configs {
			inj := injector.NewOutOfPodsInjector(spec, injector.OutOfPodsInjectorConfig{Config: config})
			injectors = append(injectors, inj)
		}
	},
}

func init() {
	outOfPodsCmd.Flags().Bool("forced", false, "If specified, NoExecute taint effect will be used (evicts existing pods), otherwise NoSchedule (default)")
}
