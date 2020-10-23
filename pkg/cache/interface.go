package cache

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/informers"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

type Interface interface {
	Get(ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer
	Add(ngvr identity.NamespacedGroupVersionResource, factory NamespacedDynamicInformerFactory) informers.GenericInformer

	FromCluster(locators []*identity.Locator) (current map[*identity.Locator]*unstructured.Unstructured)
}