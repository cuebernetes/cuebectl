package cmd

import (
	"fmt"
	"github.com/cuebernetes/cuebectl/pkg/signals"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/cuebernetes/cuebectl/pkg/cuebe"
)

var (
	applyLong = templates.LongDesc(`
		Apply cue definitions to a cluster, and unify with the cluster state.`)

	applyExample = templates.Examples(`
		# Apply a folder with cue definitions to a cluster
		%[1]s apply -f example`)
)

// ApplyOptions contains the input to the apply command.
type ApplyOptions struct {
	configFlags *genericclioptions.ConfigFlags

	CmdParent         string
	Namespace         string
	ExplicitNamespace bool

	resource.FilenameOptions
	genericclioptions.IOStreams
}

// NewApplyOptions
func NewApplyOptions(parent string, flags *genericclioptions.ConfigFlags, streams genericclioptions.IOStreams) *ApplyOptions {
	return &ApplyOptions{
		configFlags: flags,
		CmdParent:   parent,
		IOStreams:   streams,
	}
}

// NewCmdApply creates a command object for the "apply"
func NewCmdApply(parent string, flags *genericclioptions.ConfigFlags, streams genericclioptions.IOStreams) *cobra.Command {
	f := cmdutil.NewFactory(flags)
	o := NewApplyOptions(parent, flags, streams)

	cmd := &cobra.Command{
		Use:                   fmt.Sprintf("%s apply [flags]", parent),
		DisableFlagsInUseLine: true,
		Short:                 "Apply cue manifests",
		Long:                  applyLong + "\n\n" + cmdutil.SuggestAPIResources(parent),
		Example:               fmt.Sprintf(applyExample, parent),
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.Complete(f, cmd, args))
			cmdutil.CheckErr(o.Validate(cmd, args))
			cmdutil.CheckErr(o.Run(f, cmd, args))
		},
	}

	cmd.Flags().BoolP("help", "h", false, fmt.Sprintf("Help for %s apply", parent))
	cmdutil.AddFilenameOptionFlags(cmd, &o.FilenameOptions, "identifying the resources to send to a server.")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete takes the command arguments and factory and infers any remaining options.
func (o *ApplyOptions) Complete(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	var err error

	o.Namespace, o.ExplicitNamespace, err = f.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return err
	}

	return nil
}

// Validate checks the set of flags provided by the user.
func (o *ApplyOptions) Validate(cmd *cobra.Command, args []string) error {
	if len(args) == 0 && cmdutil.IsFilenameSliceEmpty(o.Filenames, o.Kustomize) {
		return fmt.Errorf("must supply a path to cue files")
	}
	return nil
}

// Run performs the apply operation.
func (o *ApplyOptions) Run(f cmdutil.Factory, cmd *cobra.Command, args []string) error {
	client, err := f.DynamicClient()
	if err != nil {
		return err
	}
	mapper, err := f.ToRESTMapper()
	if err != nil {
		return err
	}
	return cuebe.Do(signals.Context(), client, mapper, args[0])
}
