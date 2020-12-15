// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package unifier

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

type Interface interface {
	Fill(queue workqueue.RateLimitingInterface) (total int, err error)
	Lookup(fromCluster map[*identity.Locator]*unstructured.Unstructured, path ...string) (*unstructured.Unstructured, error)
}


