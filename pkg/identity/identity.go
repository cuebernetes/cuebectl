package identity

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Locator is used to track objects that have been synced, and therefore have a known GVR, Name, and Namespace
type Locator struct {
	NamespacedGroupVersionResource
	Name string
	// Path in instance
	Path []string
}

// NamespacedGroupVersionResource is used to look up informers for resolved objects from the instance
type NamespacedGroupVersionResource struct {
	schema.GroupVersionResource
	Namespace string
}
