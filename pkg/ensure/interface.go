// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package ensure

import (
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Interface interface {
	// Ensure takes an unstructured object and ensures that it is either created or updated on the cluster, returning
	// the updated object or an error.
	// It should be used for resources that have been concreted via cue instance / cluster reconciliation
	EnsureUnstructured(*unstructured.Unstructured) (*unstructured.Unstructured, identity.Locator, error)
}
