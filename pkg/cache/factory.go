// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package cache

import (
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

type NamespacedDynamicInformerFactory func(client dynamic.Interface, ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer

func DefaultNamespacedDynamicInformerFactory(client dynamic.Interface, ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer {
	return dynamicinformer.NewFilteredDynamicInformer(client, ngvr.GroupVersionResource, ngvr.Namespace, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, nil)
}

var _ NamespacedDynamicInformerFactory = DefaultNamespacedDynamicInformerFactory
