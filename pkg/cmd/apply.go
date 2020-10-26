package cmd

import (
	"context"
	"fmt"
	"strings"

	"cuelang.org/go/cue/load"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/cli-runtime/pkg/resource"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/cuebernetes/cuebectl/pkg/controller"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/signals"
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
	Watch             bool

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
	cmd.Flags().BoolP("watch", "w", false, "after creating resources, continue to watch cluster state")
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
	o.Watch, err = cmd.Flags().GetBool("watch")
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

	is := load.Instances([]string{"."}, &load.Config{
		Dir: args[0],
	})
	if len(is) > 1 {
		return fmt.Errorf("multiple instance loading currently not supported")
	}
	if len(is) < 1 {
		return fmt.Errorf("no instances found")
	}
	cueInstanceController := controller.NewCueInstanceController(client, mapper, is[0])
	stateChan := make(chan map[*identity.Locator]*unstructured.Unstructured)
	ctx, cancel := context.WithCancel(signals.Context())
	count, err := cueInstanceController.Start(ctx, stateChan)
	if err != nil {
		return err
	}

	printer := printers.NewTablePrinter(printers.PrintOptions{})
	for {
		select {
		case current := <-stateChan:
			if !o.Watch && count == len(current) {
				cancel()
			}

			if err := printer.PrintObj(toTable(current), o.IOStreams.Out); err != nil {
				return err
			}

		case <-ctx.Done():
			return nil
		default:
		}
	}
}

func toTable(current map[*identity.Locator]*unstructured.Unstructured) *metav1.Table {
	t := metav1.Table{
		TypeMeta: metav1.TypeMeta{
			Kind: "Table",
			APIVersion: "meta.k8s.io/v1beta1",
		},
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{
				Name: "path",
				Type: "string",
			},
			{
				Name: "kind",
				Type: "string",
			},
			{
				Name: "apiVersion",
				Type: "string",
			},
			{
				Name: "namespace",
				Type: "string",
			},
			{
				Name: "name",
				Type: "string",
			},
		},
		Rows:  []metav1.TableRow{},
	}
	for l, obj := range current {
		t.Rows = append(t.Rows, metav1.TableRow{
			Cells: []interface{}{
				strings.Join(l.Path, "/"),
				obj.GetKind(),
				obj.GetAPIVersion(),
				obj.GetNamespace(),
				obj.GetName(),
			},
		})
	}
	return &t
}