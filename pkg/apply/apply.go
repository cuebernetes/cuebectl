// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package apply

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/cuebernetes/cuebectl/pkg/controller"
	"github.com/cuebernetes/cuebectl/pkg/identity"
)

func CueDir(ctx context.Context, out io.Writer, client dynamic.Interface, mapper meta.RESTMapper, path string, keepalive bool) error {
	is := load.Instances([]string{"."}, &load.Config{
		Dir: path,
	})
	if len(is) > 1 {
		return fmt.Errorf("multiple instance loading currently not supported")
	}
	if len(is) < 1 {
		return fmt.Errorf("no instances found")
	}
	cueInstanceController := controller.NewCueInstanceController(client, mapper, is[0])
	stateChan := make(chan map[*identity.Locator]*unstructured.Unstructured)
	errChan := make(chan error)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	count, err := cueInstanceController.Start(ctx, stateChan, errChan)
	if err != nil {
		return err
	}

	printed := map[string]struct{}{}
	for {
		select {
		case current := <-stateChan:
			if !keepalive && count == len(current) {
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
					return err
				}
			}
		case err := <-errChan:
			if _, err := fmt.Fprintln(out, err); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		default:
		}
	}
}

