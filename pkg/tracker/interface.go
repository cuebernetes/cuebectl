// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package tracker

import (
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type Interface interface {
	Sync(obj *unstructured.Unstructured, path ...string) (string, *identity.Locator, error)
	Locators() (locators []*identity.Locator)
}
