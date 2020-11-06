// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package ensure

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/davecgh/go-spew/spew"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

const ObjectHashKey = "cuebectl/object-hash"

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

func (e *DynamicUnstructuredEnsurer) EnsureUnstructured(in *unstructured.Unstructured) (out *unstructured.Unstructured, locator identity.Locator, err error) {
	gvk := in.GroupVersionKind()
	mapping, err := e.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return
	}

	// set hash of incoming object as an annotation
	err = HashUnstructured(in)
	if err != nil {
		// unexpected hash error
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

	existing, err := e.client.Resource(mapping.Resource).Namespace(namespace).Get(context.TODO(), in.GetName(), v1.GetOptions{})
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

	if EqualHash(in, existing) {
		klog.V(4).Infof("input hash equal to existing hash, no work to do")
		return
	}

	// apply if exists
	b, err := in.MarshalJSON()
	if err != nil {
		return
	}

	// TODO: is force needed (check kube versions)
	force := true
	out, err = e.client.Resource(mapping.Resource).Namespace(namespace).Patch(context.TODO(), in.GetName(), types.ApplyPatchType, b, v1.PatchOptions{FieldManager: "cuebectl", Force: &force})
	return
}

// HashUnstructured writes specified object to hash using the spew library
// which follows pointers and prints actual values of the nested objects
// ensuring the hash does not change when a pointer changes.
func HashUnstructured(u *unstructured.Unstructured) error {
	// remove hash annotation before hashing
	if _, ok := u.GetAnnotations()[ObjectHashKey]; ok {
		a := u.GetAnnotations()
		delete(a, ObjectHashKey)
		u.SetAnnotations(a)
	}
	u.SetResourceVersion("")
	u.SetManagedFields(nil)

	hasher := fnv.New32a()
	hasher.Reset()
	printer := spew.ConfigState{
		Indent:         " ",
		SortKeys:       true,
		DisableMethods: true,
		SpewKeys:       true,
	}
	_, err := printer.Fprintf(hasher, "%#v", u)
	if err != nil {
		return err
	}

	annotations := u.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[ObjectHashKey] = rand.SafeEncodeString(fmt.Sprint(hasher.Sum32()))
	u.SetAnnotations(annotations)

	return nil
}

func EqualHash(in, existing *unstructured.Unstructured) bool {
	inAnnotations := in.GetAnnotations()
	if inAnnotations == nil {
		return false
	}
	existingAnnotations := existing.GetAnnotations()
	if existingAnnotations == nil {
		return false
	}
	inHash, ok := inAnnotations[ObjectHashKey]
	if !ok {
		return false
	}
	existingHash, ok := existingAnnotations[ObjectHashKey]
	if !ok {
		return false
	}
	klog.V(4).Infof("input hash: %s, existing hash: %s", inHash, existingHash)
	return inHash == existingHash
}
