// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package apply

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue/load"
	"cuelang.org/go/cue"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"

	"github.com/cuebernetes/cuebectl/pkg/controller"
)

func CueDir(ctx context.Context, out io.Writer, client dynamic.Interface, mapper meta.RESTMapper, path string, watch bool) (*controller.ClusterState, error) {
	is := load.Instances([]string{"."}, &load.Config{
		Dir: path,
	})
	if len(is) > 1 {
		return nil, fmt.Errorf("multiple instance loading currently not supported")
	}
	if len(is) < 1 {
		return nil, fmt.Errorf("no instances found")
	}
	r := cue.Runtime{}
	instance, err := r.Build(is[0])
	if err != nil {
		return nil, err
	}
	return CueInstance(ctx, out, client, mapper, &r, instance, watch)
}

func CueInstance(ctx context.Context, out io.Writer, client dynamic.Interface, mapper meta.RESTMapper, runtime *cue.Runtime, instance *cue.Instance, watch bool) (*controller.ClusterState, error) {
	cueInstanceController := controller.NewCueInstanceController(client, mapper, runtime, instance)
	stateChan := make(chan controller.ClusterState)
	errChan := make(chan error)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	count, err := cueInstanceController.Start(ctx, stateChan, errChan)
	if err != nil {
		return nil, err
	}

	var lastState *controller.ClusterState
	printed := map[string]struct{}{}
	for {
		select {
		case current := <-stateChan:
			lastState = &current
			if !watch && count == len(current) {
				cancel()
			}
			for l, u := range current {
				path := strings.Join(l.Path, "/")
				if _, ok := printed[path]; ok {
					continue
				}
				printed[path] = struct{}{}
				if _, err := fmt.Fprintf(out,
					"created %s: %s/%s (%s)\n",
					path, u.GetNamespace(), u.GetName(), u.GroupVersionKind()); err != nil {
					return nil, err
				}
			}
		case err := <-errChan:
			if _, err := fmt.Fprintln(out, err); err != nil {
				return nil, err
			}
		case <-ctx.Done():
			return lastState, nil
		default:
		}
	}
}

