package cuebe

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/accumulator"
	"github.com/cuebernetes/cuebectl/pkg/cache"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/unifier"
)

func Do(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, path string, watch bool) error {
	doctx, cancel := context.WithCancel(ctx)

	var r cue.Runtime

	ensurer := ensure.NewDynamicUnstructuredEnsurer(client, mapper)
	informerCache := cache.NewDynamicInformerCache(client, doctx.Done())

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

	clusterUnifier, err := unifier.NewClusterUnifier(&r, is[0], informerCache, doctx.Done())
	if err != nil {
		return err
	}

	syncer := accumulator.NewLocationAccumulator(ensurer)

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

	clusterQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	sync := func() {
		for {
			if clusterQueue.ShuttingDown() {
				return
			}

			func() {
				item, shutdown := clusterQueue.Get()
				if shutdown {
					return
				}
				defer clusterQueue.Done(item)

				u, ok := item.(*identity.LocatedUnstructured)
				if !ok {
					fmt.Printf("expected object of type LocatedUnstructured, got: %#v\n", u)
					return
				}
				queue.Add(u.Locator.Path[0])
				fmt.Println("Cluster State: ")
				fromCluster := informerCache.FromCluster(syncer.Locators())
				for _, o := range fromCluster {
					fmt.Println(o.GetObjectKind().GroupVersionKind().String(), o.GetName(), o.GetNamespace())
				}
				fmt.Println("End Cluster State")
			}()
		}
	}
	go sync()

	process := func() {
		for {
			if queue.ShuttingDown() {
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
				locators := syncer.Locators()
				if len(locators) == total && !watch {
					cancel()
					return
				}
				fromCluster := informerCache.FromCluster(locators)
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
				inf := informerCache.Get(locator.NamespacedGroupVersionResource)
				if inf == nil {
					inf = informerCache.Add(locator.NamespacedGroupVersionResource, cache.DefaultNamespacedDynamicInformerFactory)
				}
				// add an eventhandler that only reacts to the synced object
				inf.Informer().AddEventHandler(locator.EventHandler(clusterQueue))

				queue.Forget(item)
			}()
		}
	}

	go process()

	<-doctx.Done()
	return nil
}
