// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package unifier

import (
	"fmt"
	"strings"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/cache"
	"github.com/cuebernetes/cuebectl/pkg/identity"
)

// ClusterUnifier takes an initial cue.Instance and can return a new cue.Instance where initial has been unified
// with the current state of the cluster.
type ClusterUnifier struct {
	runtime     *cue.Runtime
	instance    *build.Instance
	informerSet cache.Interface

	// protects access to the build.Instance being unified
	sync.RWMutex
}

func NewClusterUnifier(instance *build.Instance, informerSet cache.Interface) *ClusterUnifier {
	return &ClusterUnifier{
		runtime:     &cue.Runtime{},
		instance:    instance,
		informerSet: informerSet,
	}
}

// unify takes the initial instance and updates the unified representation with current cluster state.
// the cluster state is constructed from the informer cache
func (u *ClusterUnifier) unify(fromCluster map[*identity.Locator]*unstructured.Unstructured) (instance *cue.Instance, err error) {
	u.Lock()
	defer u.Unlock()

	instance, err = u.runtime.Build(u.instance)
	if err != nil {
		return
	}
	for l, o := range fromCluster {
		if instance, err = instance.Fill(o, l.Path...); err != nil {
			return
		}
	}
	return
}

func (u *ClusterUnifier) Fill(queue workqueue.RateLimitingInterface) (total int, err error) {
	instance, err := u.runtime.Build(u.instance)
	if err != nil {
		return
	}

	itr, err := instance.Value().Fields()
	if err != nil {
		return
	}
	for itr.Next() {
		total++
		queue.Add(itr.Label())
	}
	return
}

// Lookup first unifies the instance with the cluster state, and then does a lookup of path in the unified instance
// if the value is concrete, the unstructured representation will be returned.
func (u *ClusterUnifier) Lookup(fromCluster map[*identity.Locator]*unstructured.Unstructured, path ...string) (*unstructured.Unstructured, error) {
	instance, err := u.unify(fromCluster)
	if err != nil {
		return nil, err
	}

	u.RLock()
	defer u.RUnlock()
	cueValue := instance.Lookup(path...)
	if err := cueValue.Validate(cue.Concrete(true)); err != nil {
		// note: err is not safe to return over the error chan because it holds references to the instance internals.
		// this takes the error string only and returns it
		return nil, fmt.Errorf("%s not yet concrete: %s", strings.Join(path, "/"), err.Error())
	}

	obj := &unstructured.Unstructured{}

	if err := cueValue.Decode(obj); err != nil {
		return nil, err
	}

	return obj, nil
}
