// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package cache

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/informers"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

type Interface interface {
	Get(ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer
	Add(ngvr identity.NamespacedGroupVersionResource, factory NamespacedDynamicInformerFactory, stopc <-chan struct{}) informers.GenericInformer

	FromCluster(locators []*identity.Locator) (current map[*identity.Locator]*unstructured.Unstructured)
}
