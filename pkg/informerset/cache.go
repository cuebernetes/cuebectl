package informerset

import (
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"

	"github.com/cuebernetes/cuebectl/pkg/identity"
)

type Interface interface {
	Get(ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer
	Add(ngvr identity.NamespacedGroupVersionResource, factory NamespacedDynamicInformerFactory) informers.GenericInformer
}

var _ Interface = &DynamicInformerSet{}

type DynamicInformerSet struct {
	client dynamic.Interface

	// the keys are write-once, so a sync.Map will work fine and reduce lock contention
	informers sync.Map

	stopc chan struct{}
}

func NewDynamicInformerSet(client dynamic.Interface, stopc chan struct{}) *DynamicInformerSet {
	return &DynamicInformerSet{
		client:    client,
		informers: sync.Map{},
		stopc:     stopc,
	}
}

func (i *DynamicInformerSet) Get(ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer {
	informer, ok := i.informers.Load(ngvr)
	if !ok {
		return nil
	}
	return informer.(informers.GenericInformer)
}

func (i *DynamicInformerSet) Add(ngvr identity.NamespacedGroupVersionResource, factory NamespacedDynamicInformerFactory) informers.GenericInformer {
	inf := factory(i.client, ngvr)
	i.informers.Store(ngvr, inf)
	go inf.Informer().Run(i.stopc)
	return inf
}

type NamespacedDynamicInformerFactory func(client dynamic.Interface, ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer

func DefaultNamespacedDynamicInformerFactory(client dynamic.Interface, ngvr identity.NamespacedGroupVersionResource) informers.GenericInformer {
	return dynamicinformer.NewFilteredDynamicInformer(client, ngvr.GroupVersionResource, ngvr.Namespace, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, nil)
}

var _ NamespacedDynamicInformerFactory = DefaultNamespacedDynamicInformerFactory
