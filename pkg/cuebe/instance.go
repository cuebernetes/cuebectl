package cuebe

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/ensure"
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
	return e.ConcreteError.Error()
}

type ClusterUnifier struct {
	initial, unified *cue.Instance

	informerSet informerset.Interface

	// locators maps a label to the object's GVR/Name/Namespace if it's been synced once, or nil if it hasn't
	locators map[string]*identity.Locator

	ensurer ensure.Interface

	stopc chan struct{}

	mu sync.Mutex
}

func (t *ClusterUnifier) Sync(label string) error {
	t.mu.Lock()
	// note: label came from initial, so we shouldn't need to check existence after lookup
	cueValue := t.unified.Lookup(label)
	t.mu.Unlock()

	if err := cueValue.Validate(cue.Concrete(true)); err != nil {
		return NotConcreteError{
			Value:         cueValue,
			ConcreteError: err,
		}
	}

	obj := &unstructured.Unstructured{}

	if err := cueValue.Decode(obj); err != nil {
		return err
	}

	_, locator, err := t.ensurer.EnsureUnstructured(obj)
	if err != nil {
		return err
	}

	t.mu.Lock()
	t.locators[label] = &locator
	t.mu.Unlock()

	inf := t.informerSet.Get(locator.NamespacedGroupVersionResource)
	if inf == nil {
		inf = t.informerSet.Add(locator.NamespacedGroupVersionResource, informerset.DefaultNamespacedDynamicInformerFactory)
		// TODO: should probably not block and instead rely on cache events
		if ok := cache.WaitForCacheSync(t.stopc, inf.Informer().HasSynced); !ok {
			fmt.Println("Error waiting for cache to Sync")
		}
	}

	return t.Unify()
}

// Unify takes the initial instance and updates the unified representation with current cluster state.
// the cluster state is constructed from the informer cache
func (t *ClusterUnifier) Unify() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.unified = t.initial
	for l, o := range t.locators {
		if o == nil {
			continue
		}

		i := t.informerSet.Get(o.NamespacedGroupVersionResource)

		var fetched runtime.Object
		var err error
		if o.Namespace != "" {
			fetched, err = i.Lister().ByNamespace(o.Namespace).Get(o.Name)
		} else {
			fetched, err = i.Lister().Get(o.Name)
		}

		// TODO: should this trigger a retry, since the unified state will be dirty?
		if err != nil {
			fmt.Println("WARNING:", l, "has been synced but not found in cache")
			continue
		}

		if t.unified, err = t.unified.Fill(fetched, l); err != nil {
			return err
		}
	}

	return nil
}

func NewClusterUnifier(instance *cue.Instance, informerSet informerset.Interface, e ensure.Interface) (*ClusterUnifier, error) {
	current := make(map[string]*identity.Locator)
	itr, err := instance.Value().Fields()
	if err != nil {
		return nil, err
	}
	for itr.Next() {
		current[itr.Label()] = nil
	}
	return &ClusterUnifier{
		initial:     instance,
		unified:     instance,
		locators:    current,
		informerSet: informerSet,
		ensurer:     e,
	}, nil
}

func Do(client dynamic.Interface, mapper meta.RESTMapper, path string) error {
	var r cue.Runtime

	stopc := make(chan struct{})
	ensurer := ensure.NewDynamicUnstructuredEnsurer(client, mapper)
	informerSet := informerset.NewDynamicInformerSet(client, stopc)

	is := load.Instances([]string{"."}, &load.Config{
		Dir: path,
	})
	if len(is) > 1 {
		return fmt.Errorf("multiple instance loading currently not supported")
	}
	if len(is) < 1 {
		return fmt.Errorf("no instances found")
	}

	instance, err := r.Build(is[0])
	if err != nil {
		return err
	}

	unifier, err := NewClusterUnifier(instance, informerSet, ensurer)
	if err != nil {
		return err
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	for l := range unifier.locators {
		queue.Add(l)
	}

	var wg sync.WaitGroup
	process := func() {
		for {
			if queue.ShuttingDown() {
				wg.Done()
				break
			}

			func() {
				item, shutdown := queue.Get()
				if shutdown {
					return
				}
				defer queue.Done(item)

				label, ok := item.(string)
				if !ok {
					fmt.Printf("WARNING: %#v is being dropped because it is not a string\n", label)
					queue.Forget(label)
					return
				}

				err := unifier.Sync(label)
				var notConcreteError NotConcreteError
				if errors.As(err, &notConcreteError) {
					fmt.Println(label, "cannot be created yet: ", err)
					queue.AddRateLimited(item)
					return
				}
				if err != nil {
					fmt.Printf("error syncing %v with cluster: %s\n", item, err)
					queue.AddRateLimited(item)
					return
				}

				fmt.Println("created", label)

				queue.Forget(item)
			}()
		}
	}

	wg.Add(2)
	go process()
	go process()

	// shut down queue only when everything has been synced
	go func() {
		for {
			incomplete := false

			unifier.mu.Lock()
			for _, s := range unifier.locators {
				if s == nil {
					incomplete = true
					break
				}
			}
			unifier.mu.Unlock()

			if incomplete {
				time.Sleep(time.Second)
			} else {
				queue.ShutDown()
				return
			}
		}
	}()

	wg.Wait()

	return nil
}
