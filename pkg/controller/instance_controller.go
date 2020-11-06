// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package controller

import (
	"context"

	"cuelang.org/go/cue/build"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/cuebernetes/cuebectl/pkg/cache"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
	"github.com/cuebernetes/cuebectl/pkg/tracker"
	"github.com/cuebernetes/cuebectl/pkg/unifier"
)

type CueInstanceController struct {
	clusterQueue, cueQueue workqueue.RateLimitingInterface
	informerCache          *cache.DynamicInformerCache
	tracker                *tracker.LocationTracker
	unifier                *unifier.ClusterUnifier
}

func NewCueInstanceController(client dynamic.Interface, mapper meta.RESTMapper, buildInstance *build.Instance) *CueInstanceController {
	informerCache := cache.NewDynamicInformerCache(client)
	return &CueInstanceController{
		clusterQueue:  workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		cueQueue:      workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		tracker:       tracker.NewLocationTracker(ensure.NewDynamicUnstructuredEnsurer(client, mapper, informerCache)),
		unifier:       unifier.NewClusterUnifier(buildInstance, informerCache),
		informerCache: informerCache,
	}
}

func (c *CueInstanceController) Start(ctx context.Context, stateChan chan map[*identity.Locator]*unstructured.Unstructured, errChan chan error) (count int, err error) {
	count, err = c.unifier.Fill(c.cueQueue)
	go c.processClusterStateQueue(stateChan)
	go c.processCueQueue(ctx, errChan)
	return
}

func (c *CueInstanceController) syncUnstructured(u *identity.LocatedUnstructured, stateChan chan map[*identity.Locator]*unstructured.Unstructured) {
	// requeue the label associated with the object in the cue instance
	c.cueQueue.Add(u.Locator.Path[0])

	// send back current cluster state
	stateChan <- c.informerCache.FromCluster(c.tracker.Locators())
}

func (c *CueInstanceController) syncCueInstance(label string, errChan chan error, stopc <-chan struct{}) {
	// unify cue instance with current cluster state and lookup value at `label`
	obj, err := c.unifier.Lookup(c.informerCache.FromCluster(c.tracker.Locators()), label)
	if err != nil {
		errChan <- err
		klog.V(1).Error(err, "could not lookup")
		c.cueQueue.AddRateLimited(label)
		return
	}

	// sync value at `label` with the cluster
	locator, err := c.tracker.Sync(obj, label)
	if err != nil {
		errChan <- err
		klog.V(1).Error(err, "could not sync")
		c.cueQueue.AddRateLimited(label)
		return
	}

	// start up informers for newly synced NGVRs
	inf := c.informerCache.Get(locator.NamespacedGroupVersionResource)
	if inf == nil {
		inf = c.informerCache.Add(locator.NamespacedGroupVersionResource, cache.DefaultNamespacedDynamicInformerFactory, stopc)
	}
	// add an eventhandler that only reacts to the synced object
	inf.Informer().AddEventHandler(locator.EventHandler(c.clusterQueue))

	c.cueQueue.Forget(label)
}

func (c *CueInstanceController) processClusterStateQueue(stateChan chan map[*identity.Locator]*unstructured.Unstructured) {
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
				klog.V(2).Infof("expected object of type LocatedUnstructured, got: %#v\n", u)
				return
			}
			c.syncUnstructured(u, stateChan)
		}()
	}
}

func (c *CueInstanceController) processCueQueue(ctx context.Context, errChan chan error) {
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
				klog.V(2).Infof("expected string, got: %#v\n", label)
				c.cueQueue.Forget(label)
				return
			}
			c.syncCueInstance(label, errChan, ctx.Done())
		}()
	}
}
