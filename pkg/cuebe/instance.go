package cuebe

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"
)

func Do(client dynamic.Interface, mapper meta.RESTMapper, path string) error {
	var r cue.Runtime

	ensurer := ensure.NewDynamicUnstructuredEnsurer(client, mapper)

	is := load.Instances([]string{"."}, &load.Config{
		Dir: path,
	})

	var instance *cue.Instance
	for _, i := range is {
		if instance == nil {
			i2, err := r.Build(i)
			if err != nil {
				return err
			}
			instance = i2
		}
		instance.Build(i)
	}

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter())
	itr, err := instance.Value().Fields()
	if err != nil {
		return err
	}
	for itr.Next() {
		queue.Add(itr.Label())
	}
	for {
		if queue.Len() == 0 {
			queue.ShutDown()
			break
		}

		func () {
			item, shutdown := queue.Get()
			if shutdown {
				return
			}
			defer queue.Done(item)

			label, ok :=  item.(string)
			if !ok {
				fmt.Printf("WARNING: %#v is being dropped because it is not a string\n", label)
				queue.Forget(label)
				return
			}
			cueValue := instance.Lookup(label)
			if !cueValue.Exists() {
				fmt.Printf("WARNING: %s does not exist in the instance\n", label)
				queue.Forget(label)
				return
			}

			if err := cueValue.Validate(cue.Concrete(true)); err != nil {
				fmt.Println(label, "cannot be created yet: ", err)
				queue.AddRateLimited(item)
				return
			}

			instance, err = instanceFromCluster(cueValue, label, instance, ensurer)
			if err != nil {
				fmt.Printf("error syncing %v with cluster", item)
				queue.AddRateLimited(item)
				return
			}

			fmt.Println("created", label)

			queue.Forget(item)
		}()
	}

	return nil
}

// instanceFromCluster returns a new instance with the value of in from the cluster by creating in
func instanceFromCluster(in cue.Value, label string, i *cue.Instance, ensurer ensure.Interface) (*cue.Instance, error) {
	obj := &unstructured.Unstructured{}

	if err := in.Decode(obj); err != nil {
		return nil, err
	}
	out, err := ensurer.EnsureUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return i.Fill(out, label)
}
