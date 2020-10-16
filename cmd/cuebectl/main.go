package main

import (
	"os"

	"github.com/cuebernetes/cuebectl/pkg/cmd"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func main() {
	flags := genericclioptions.NewConfigFlags(true)
	streams := genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr}
	root := &cobra.Command{
		Use:     "cuebectl",
		Short:   "a tool for interacting with kube clusters via cue manifests",
		//Version: version.Version,
	}
	root.AddCommand(cmd.NewCmdApply("apply", flags, streams))

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
