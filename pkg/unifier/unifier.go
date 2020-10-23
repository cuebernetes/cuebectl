package unifier

import (
	"fmt"
	"sync"

	"cuelang.org/go/cue"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/informerset"
)

// NotConcreteError is returned if a cue.Value is locators that is not yet complete
type NotConcreteError struct {
	Value         cue.Value
	ConcreteError error
}

// Error returns the underlying error indicating why the value is not concrete
func (e NotConcreteError) Error() string {
	label, _ := e.Value.Label()
	return errors.WithMessagef(e.ConcreteError, "%s not yet concrete", label).Error()
}

// ClusterUnifier takes an initial cue.Instance and can return a new cue.Instance where initial has been unified
// with the current state of the cluster.
type ClusterUnifier struct {
	initial, unified *cue.Instance
	informerSet      informerset.Interface
	stopc            <-chan struct{}

	// protects access to the cue.Instance being unified
	sync.RWMutex
}

// unify takes the initial instance and updates the unified representation with current cluster state.
// the cluster state is constructed from the informer cache
func (u *ClusterUnifier) unify(fromCluster map[*identity.Locator]runtime.Object) (err error) {
	u.Lock()
	defer u.Unlock()

	u.unified = u.initial
	for l, o := range fromCluster {
		if u.unified, err = u.unified.Fill(o, l.Path...); err != nil {
			return
		}
	}
	return
}

// Lookup first unifies the instance with the cluster state, and then does a lookup of path in the unified instance
// if the value is concrete, the unstructured representation will be returned.
func (u *ClusterUnifier) Lookup(fromCluster map[*identity.Locator]runtime.Object, path ...string) (*unstructured.Unstructured, error) {
	if err := u.unify(fromCluster); err != nil {
		return nil, err
	}
	u.RLock()
	defer u.RUnlock()

	cueValue := u.unified.Lookup(path...)
	if err := cueValue.Validate(cue.Concrete(true)); err != nil {
		return nil, NotConcreteError{
			Value:         cueValue,
			ConcreteError: err,
		}
	}

	obj := &unstructured.Unstructured{}

	if err := cueValue.Decode(obj); err != nil {
		return nil, err
	}

	return obj, nil
}

// FromCluster returns a list of objects found in the cluster (cache) identified by locators
func (u *ClusterUnifier) FromCluster(locators []*identity.Locator) (current map[*identity.Locator]runtime.Object) {
	current = make(map[*identity.Locator]runtime.Object)

	for _, o := range locators {
		i := u.informerSet.Get(o.NamespacedGroupVersionResource)

		var fetched runtime.Object
		var err error
		if o.Namespace != "" {
			fetched, err = i.Lister().ByNamespace(o.Namespace).Get(o.Name)
		} else {
			fetched, err = i.Lister().Get(o.Name)
		}

		// TODO: should this trigger a retry, since the unified state will be dirty?
		if err != nil {
			fmt.Println("WARNING:", o.Path, "has been synced but not found in cache")
			continue
		}
		current[o] = fetched
	}

	return
}

func NewClusterUnifier(instance *cue.Instance, informerSet informerset.Interface, stopc <-chan struct{}) (*ClusterUnifier, error) {
	return &ClusterUnifier{
		initial:     instance,
		informerSet: informerSet,
		stopc:       stopc,
	}, nil
}

