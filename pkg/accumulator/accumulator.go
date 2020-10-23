package accumulator

import (
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"github.com/cuebernetes/cuebectl/pkg/identity"
)

// LocationAccumulator tracks locations (in a cluster) of objects that have been created / synced from an instance.
type LocationAccumulator struct {
	ensurer ensure.Interface

	// returns locators after they have been synced the first time.
	synced chan identity.Locator

	// locators to lookup values that have already been synced at least once
	locators []*identity.Locator

	sync.RWMutex
}

func NewLocationAccumulator(ensurer ensure.Interface, synced chan identity.Locator) *LocationAccumulator {
	return &LocationAccumulator{
		ensurer:  ensurer,
		synced:   synced,
		locators: make([]*identity.Locator, 0),
	}
}

// Sync attempts to create an unstructured object identified by []path in instance.
// if successful, it returns a locator that can be used to lookup the object in the cluster later.
func (a *LocationAccumulator) Sync(obj *unstructured.Unstructured, path ...string) (*identity.Locator, error) {
	_, locator, err := a.ensurer.EnsureUnstructured(obj)
	if err != nil {
		return nil, err
	}
	locator.Path = path
	a.synced <- locator

	a.Lock()
	a.locators = append(a.locators, &locator)
	a.Unlock()
	return &locator, nil
}

// Locators returns the list of locators for concrete values
func (a *LocationAccumulator) Locators() (locators []*identity.Locator) {
	locators = make([]*identity.Locator, 0)
	a.RLock()
	defer a.RUnlock()
	locators = append(locators, a.locators...)
	return
}
