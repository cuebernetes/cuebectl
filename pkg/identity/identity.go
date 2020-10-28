// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package identity

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// NamespacedGroupVersionResource is used to look up informers for resolved objects from the instance
type NamespacedGroupVersionResource struct {
	schema.GroupVersionResource
	Namespace string
}

// Locator is used to track objects that have been synced, and therefore have a known GVR, Name, and Namespace
type Locator struct {
	NamespacedGroupVersionResource
	Name string
	// Path in instance
	Path []string
}

type LocatedUnstructured struct {
	Locator
	*unstructured.Unstructured
}

// FilterFunc returns a function that can filter events to only react to objects identified by the locator
func (l Locator) FilterFunc() func(o interface{}) bool {
	return func(o interface{}) bool {
		u, ok := o.(*unstructured.Unstructured)
		if !ok {
			return false
		}
		return u.GetName() == l.Name && u.GetNamespace() == l.Namespace
	}
}

// EventHandler returns an event handler for this locator that adds a locator to the incoming object and adds to queue
func (l Locator) EventHandler(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	addToQueue := func(obj interface{}) {
		queue.Add(&LocatedUnstructured{
			Locator:      l,
			Unstructured: obj.(*unstructured.Unstructured),
		})
	}
	return cache.FilteringResourceEventHandler{
		FilterFunc: l.FilterFunc(),
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				addToQueue(obj)
			},
			UpdateFunc: func(oldObj, obj interface{}) {
				addToQueue(obj)
			},
			DeleteFunc: func(obj interface{}) {
				addToQueue(obj)
			},
		},
	}
}
