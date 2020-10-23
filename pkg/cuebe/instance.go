package cuebe

import (
	"context"
	"fmt"
	"sync"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/accumulator"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/informerset"
	"github.com/cuebernetes/cuebectl/pkg/unifier"
)

func Do(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, path string) error {
	var r cue.Runtime

	ensurer := ensure.NewDynamicUnstructuredEnsurer(client, mapper)
	informerSet := informerset.NewDynamicInformerSet(client, ctx.Done())

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
	clusterUnifier, err := unifier.NewClusterUnifier(instance, informerSet, ctx.Done())
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

	clusterQueue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func (obj interface{}) {
			clusterQueue.Add(obj)
		},
		UpdateFunc: func (oldObj, obj interface{}) {
			clusterQueue.Add(obj)
		},
		DeleteFunc: func (obj interface{}) {
			clusterQueue.Add(obj)
		},
	}

	watch := func() {
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

				fmt.Println("Cluster State: ")
				fromCluster := clusterUnifier.FromCluster(syncer.Locators())
				for _, o := range fromCluster {
					u := o.(*unstructured.Unstructured)
					fmt.Println(u.GetObjectKind().GroupVersionKind().String(), u.GetName(), u.GetNamespace())
				}
				fmt.Println("End Cluster State")
			}()
		}
	}
	go watch()

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
				inf := informerSet.Get(locator.NamespacedGroupVersionResource)
				if inf == nil {
					inf = informerSet.Add(locator.NamespacedGroupVersionResource, informerset.DefaultNamespacedDynamicInformerFactory)
				}
				// add an eventhandler that only reacts to the synced object
				inf.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
					FilterFunc: locator.FilterFunc(),
					Handler: handlers,
				})

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
			case p := <-syncChan:
				fmt.Println("created", p.Path)
				count++
			}
			if count == total {
				queue.ShutDown()
				return
			}
		}
	}()

	wg.Wait()

	<-ctx.Done()
	return nil
}
