package identity

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Locator is used to track objects that have been synced, and therefore have a known GVR, Name, and Namespace
type Locator struct {
	NamespacedGroupVersionResource
	Name string
	// Path in instance
	Path []string
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

// NamespacedGroupVersionResource is used to look up informers for resolved objects from the instance
type NamespacedGroupVersionResource struct {
	schema.GroupVersionResource
	Namespace string
}
