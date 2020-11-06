// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package cache

import (
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/informers"
	"k8s.io/klog/v2"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

var _ Interface = &DynamicInformerCache{}

type DynamicInformerCache struct {
	client dynamic.Interface

	// the keys are write-once, so a sync.Map will work fine and reduce lock contention
	informers sync.Map
}

func NewDynamicInformerCache(client dynamic.Interface) *DynamicInformerCache {
	return &DynamicInformerCache{
		client:    client,
		informers: sync.Map{},
	}
}

func (d *DynamicInformerCache) Get(ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer {
	informer, ok := d.informers.Load(ngvr)
	if !ok {
		return nil
	}
	return informer.(informers.GenericInformer)
}

func (d *DynamicInformerCache) Add(ngvr identity.NamespacedGroupVersionResource, factory NamespacedDynamicInformerFactory, stopc <-chan struct{}) informers.GenericInformer {
	inf := factory(d.client, ngvr)
	d.informers.Store(ngvr, inf)
	go inf.Informer().Run(stopc)
	return inf
}

// FromCluster returns a list of objects found in the cluster (cache) identified by locators
func (d *DynamicInformerCache) FromCluster(locators []*identity.Locator) (current map[*identity.Locator]*unstructured.Unstructured) {
	current = make(map[*identity.Locator]*unstructured.Unstructured)

	for _, o := range locators {
		i := d.Get(o.NamespacedGroupVersionResource)

		var fetched runtime.Object
		var err error
		if o.Namespace != "" {
			fetched, err = i.Lister().ByNamespace(o.Namespace).Get(o.Name)
		} else {
			fetched, err = i.Lister().Get(o.Name)
		}

		// returned cluster state will be dirty, but a future sync will catch up
		if err != nil {
			klog.V(2).Infof("%s has been synced but not found in cache, cluster state is dirty", strings.Join(o.Path, "/"))
			continue
		}
		u, ok := fetched.(*unstructured.Unstructured)
		if !ok {
			continue
		}

		current[o] = u
	}

	return
}
