package ensure

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

// DynamicUnstructuredEnsurer uses a dynamic client to provide ensure.Interface
type DynamicUnstructuredEnsurer struct {
	client dynamic.Interface
	mapper meta.RESTMapper
}

var _ Interface = &DynamicUnstructuredEnsurer{}

// NewDynamicUnstructuredEnsurer constructs a an ensurer from a dynamic.Interface and RESTMapper
func NewDynamicUnstructuredEnsurer(client dynamic.Interface, mapper meta.RESTMapper) *DynamicUnstructuredEnsurer {
	return &DynamicUnstructuredEnsurer{
		client: client,
		mapper: mapper,
	}
}

// TODO: add hash annotation
func (e *DynamicUnstructuredEnsurer) EnsureUnstructured(in *unstructured.Unstructured) (out *unstructured.Unstructured, locator identity.Locator, err error) {
	gvk := in.GroupVersionKind()
	mapping, err := e.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return
	}

	namespace := in.GetNamespace()

	// create if no name
	if in.GetName() == "" {
		out, err = e.client.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), in, v1.CreateOptions{FieldManager: "cuebectl"})
		if err == nil && out != nil {
			locator = identity.Locator{NamespacedGroupVersionResource: identity.NamespacedGroupVersionResource{GroupVersionResource: mapping.Resource, Namespace: namespace}, Name: out.GetName()}
		}
		return
	}

	_, err = e.client.Resource(mapping.Resource).Namespace(namespace).Get(context.TODO(), in.GetName(), v1.GetOptions{})
	if errors.IsNotFound(err) {
		// create if not exists
		out, err = e.client.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), in, v1.CreateOptions{FieldManager: "cuebectl"})
		if err == nil && out != nil {
			locator = identity.Locator{NamespacedGroupVersionResource: identity.NamespacedGroupVersionResource{GroupVersionResource: mapping.Resource, Namespace: namespace}, Name: out.GetName()}
		}
		return
	}
	if err != nil {
		// unexpected error
		return
	}

	locator = identity.Locator{NamespacedGroupVersionResource: identity.NamespacedGroupVersionResource{GroupVersionResource: mapping.Resource, Namespace: namespace}, Name: in.GetName()}

	// TODO: should this be needed?
	in.SetManagedFields(nil)

	// apply if exists
	b, err := in.MarshalJSON()
	if err != nil {
		return
	}
	out, err = e.client.Resource(mapping.Resource).Namespace(namespace).Patch(context.TODO(), in.GetName(), types.ApplyPatchType, b, v1.PatchOptions{FieldManager: "cuebectl"})
	return
}
