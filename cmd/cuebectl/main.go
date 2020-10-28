// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/component-base/cli/globalflag"

	"github.com/cuebernetes/cuebectl/pkg/cmd"
)

func main() {
	flags := genericclioptions.NewConfigFlags(true)
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	root := &cobra.Command{
		Use:   commandName(),
		Short: "a tool for interacting with kube clusters via cue manifests",
		//Version: version.Version,
	}
	globalflag.AddGlobalFlags(root.PersistentFlags(), commandName())
	root.AddCommand(cmd.NewCmdApply(commandName(), flags, streams))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func commandName() string {
	cli := filepath.Base(os.Args[0])
	if strings.HasPrefix(cli, "kubectl-") {
		return strings.TrimPrefix(cli, "kubectl-")
	}
	return cli
}
