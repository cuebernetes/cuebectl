package controller

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"

	"github.com/cuebernetes/cuebectl/pkg/cache"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/tracker"
	"github.com/cuebernetes/cuebectl/pkg/unifier"
)

type CueInstanceController struct {
	clusterQueue, cueQueue workqueue.RateLimitingInterface
	informerCache          *cache.DynamicInformerCache
	syncer                 *tracker.LocationTracker
	unifier                *unifier.ClusterUnifier

	// TODO: clean up
	context context.Context
	total   int
	watch   bool
	cancel  context.CancelFunc
}

func NewCueInstanceController(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, buildInstance *build.Instance, watch bool) (*CueInstanceController, error) {
	var r cue.Runtime
	doctx, cancel := context.WithCancel(ctx)

	ensurer := ensure.NewDynamicUnstructuredEnsurer(client, mapper)
	informerCache := cache.NewDynamicInformerCache(client, doctx.Done())

	instance, err := r.Build(buildInstance)
	if err != nil {
		return nil, err
	}
	instance.Value().Len()

	clusterUnifier, err := unifier.NewClusterUnifier(&r, buildInstance, informerCache, doctx.Done())
	if err != nil {
		return nil, err
	}

	syncer := tracker.NewLocationTracker(ensurer)

	// TODO: cleanup
	total := 0
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	itr, err := instance.Value().Fields()
	if err != nil {
		return nil, err
	}
	for itr.Next() {
		queue.Add(itr.Label())
		total++
	}
	return &CueInstanceController{
		context:       doctx,
		clusterQueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		cueQueue:      queue,
		syncer:        syncer,
		unifier:       clusterUnifier,
		informerCache: informerCache,
		total:         total,
		watch:         watch,
		cancel:        cancel,
	}, nil
}

func (c *CueInstanceController) Start() {
	go c.processClusterState()
	go c.processCueInstance()
	<-c.context.Done()
}

func (c *CueInstanceController) processClusterState() {
	for {
		if c.clusterQueue.ShuttingDown() {
			return
		}

		func() {
			item, shutdown := c.clusterQueue.Get()
			if shutdown {
				return
			}
			defer c.clusterQueue.Done(item)

			u, ok := item.(*identity.LocatedUnstructured)
			if !ok {
				fmt.Printf("expected object of type LocatedUnstructured, got: %#v\n", u)
				return
			}
			c.cueQueue.Add(u.Locator.Path[0])

			fmt.Println("Cluster State: ")
			fromCluster := c.informerCache.FromCluster(c.syncer.Locators())
			for _, o := range fromCluster {
				fmt.Println(o.GetObjectKind().GroupVersionKind().String(), o.GetName(), o.GetNamespace())
			}
			fmt.Println("End Cluster State")
		}()
	}
}

func (c *CueInstanceController) processCueInstance() {
	for {
		if c.cueQueue.ShuttingDown() {
			break
		}

		func() {
			item, shutdown := c.cueQueue.Get()
			if shutdown {
				return
			}
			defer c.cueQueue.Done(item)

			label, ok := item.(string)
			if !ok {
				fmt.Printf("WARNING: %#v is being dropped because it is not a string\n", label)
				c.cueQueue.Forget(label)
				return
			}
			locators := c.syncer.Locators()
			if len(locators) == c.total && !c.watch {
				c.cancel()
				return
			}
			fromCluster := c.informerCache.FromCluster(locators)
			obj, err := c.unifier.Lookup(fromCluster, label)
			if err != nil {
				fmt.Println(err)
				c.cueQueue.AddRateLimited(item)
				return
			}

			locator, err := c.syncer.Sync(obj, label)
			if err != nil {
				fmt.Println(err)
				c.cueQueue.AddRateLimited(item)
				return
			}

			// start up informers for newly synced NGVRs
			inf := c.informerCache.Get(locator.NamespacedGroupVersionResource)
			if inf == nil {
				inf = c.informerCache.Add(locator.NamespacedGroupVersionResource, cache.DefaultNamespacedDynamicInformerFactory)
			}
			// add an eventhandler that only reacts to the synced object
			inf.Informer().AddEventHandler(locator.EventHandler(c.clusterQueue))

			c.cueQueue.Forget(item)
		}()
	}
}
