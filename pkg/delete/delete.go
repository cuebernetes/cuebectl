// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package delete

import (
	"context"
	"fmt"
	"io"
	"strings"

	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

func All(ctx context.Context, out io.Writer, client dynamic.Interface, locators []identity.Locator, options metav1.DeleteOptions) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, l := range locators {
		l := l
		g.Go(func() (err error) {
			err = client.Resource(l.GroupVersionResource).Namespace(l.Namespace).Delete(ctx, l.Name, options)
			_, err = fmt.Fprintf(out,
				"deleted %s: %s/%s (%s)\n",
				strings.Join(l.Path, "/"), l.Namespace, l.Name, l.GroupVersionResource)
			return
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
