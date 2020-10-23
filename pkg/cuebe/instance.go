package cuebe

import (
	"fmt"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/accumulator"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/informerset"
	"github.com/cuebernetes/cuebectl/pkg/unifier"
)

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

	syncChan := make(chan identity.Locator)
	clusterUnifier, err := unifier.NewClusterUnifier(instance, informerSet, make(chan struct{}))
	if err != nil {
		return err
	}

	syncer := accumulator.NewLocationAccumulator(ensurer, syncChan)

	total := 0
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	itr, err := instance.Value().Fields()
	if err != nil {
		return err
	}
	for itr.Next() {
		queue.Add(itr.Label())
		total++
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
				fromCluster := clusterUnifier.FromCluster(syncer.Locators())
				obj, err := clusterUnifier.Lookup(fromCluster, label)
				if err != nil {
					fmt.Println(err)
					queue.AddRateLimited(item)
					return
				}

				locator, err := syncer.Sync(obj, label)
				if err != nil {
					fmt.Println(err)
					queue.AddRateLimited(item)
					return
				}

				// start up informers for newly synced NGVRs
				if inf := informerSet.Get(locator.NamespacedGroupVersionResource); inf == nil {
					informerSet.Add(locator.NamespacedGroupVersionResource, informerset.DefaultNamespacedDynamicInformerFactory)
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
		count := 0
		for {
			select {
			case <-syncChan:
				count++
			}
			if count == total {
				queue.ShutDown()
				return
			}
		}
	}()

	wg.Wait()

	return nil
}
