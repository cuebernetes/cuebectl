// SPDX-License-Identifier:  Apache-2.0
// SPDX-FileCopyrightText: 2020 Evan Cordell

package controller

import (
	"context"
	"strings"
	"sync"

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

type ClusterState map[*identity.Locator]*unstructured.Unstructured

func (c ClusterState) Locators() []identity.Locator {
	locators := make([]identity.Locator, 0)
	for l := range c {
		locators = append(locators, *l)
	}
	return locators
}

type CueInstanceController struct {
	clusterQueue, cueQueue workqueue.RateLimitingInterface
	informerCache          cache.Interface
	tracker                tracker.Interface
	unifier                unifier.Interface
	resourceVersions       *lastResourceVersions
}

func NewCueInstanceController(client dynamic.Interface, mapper meta.RESTMapper, buildInstance *build.Instance) *CueInstanceController {
	informerCache := cache.NewDynamicInformerCache(client)
	return &CueInstanceController{
		clusterQueue:     workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		cueQueue:         workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		tracker:          tracker.NewLocationTracker(ensure.NewDynamicUnstructuredEnsurer(client, mapper, informerCache)),
		unifier:          unifier.NewClusterUnifier(buildInstance, informerCache),
		informerCache:    informerCache,
		resourceVersions: NewLastResourceVersions(),
	}
}

func (c *CueInstanceController) Start(ctx context.Context, stateChan chan ClusterState, errChan chan error) (count int, err error) {
	count, err = c.unifier.Fill(c.cueQueue)
	go c.processClusterStateQueue(stateChan)
	go c.processCueQueue(ctx, errChan)
	return
}

func (c *CueInstanceController) syncUnstructured(u *identity.LocatedUnstructured, stateChan chan ClusterState) {
	if rv, ok := c.resourceVersions.Get(strings.Join(u.Locator.Path, "/")); ok && rv == u.GetResourceVersion() {
		klog.V(2).Infof("cache hasn't yet caught up to recent changes")
		return
	}

	// requeue the label associated with the object in the cue instance
	c.cueQueue.Add(strings.Join(u.Locator.Path, "/"))

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

	rv, ok := c.resourceVersions.Get(label)
	objrv := obj.GetResourceVersion()
	if ok && rv == objrv {
		klog.V(2).Infof("cache hasn't yet caught up to recent changes")
		return
	}

	// sync value at `label` with the cluster
	oldrv, locator, err := c.tracker.Sync(obj, label)
	if err != nil {
		errChan <- err
		klog.V(1).Error(err, "could not sync")
		c.cueQueue.AddRateLimited(label)
		return
	}
	c.resourceVersions.Set(label, oldrv)

	// start up informers for newly synced NGVRs
	inf := c.informerCache.Get(locator.NamespacedGroupVersionResource)
	if inf == nil {
		inf = c.informerCache.Add(locator.NamespacedGroupVersionResource, cache.DefaultNamespacedDynamicInformerFactory, stopc)
	}
	// add an eventhandler that only reacts to the synced object
	inf.Informer().AddEventHandler(locator.EventHandler(c.clusterQueue))

	c.cueQueue.Forget(label)
}

func (c *CueInstanceController) processClusterStateQueue(stateChan chan ClusterState) {
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

type lastResourceVersions struct {
	oldResourceVersions map[string]string
	sync.RWMutex
}

func NewLastResourceVersions() *lastResourceVersions {
	return &lastResourceVersions{
		oldResourceVersions: map[string]string{},
	}
}

func (t *lastResourceVersions) Set(label, rv string) {
	t.Lock()
	defer t.Unlock()
	t.oldResourceVersions[label] = rv
}

func (t *lastResourceVersions) Get(label string) (string, bool) {
	t.RLock()
	defer t.RUnlock()
	rv, ok := t.oldResourceVersions[label]
	return rv, ok
}
