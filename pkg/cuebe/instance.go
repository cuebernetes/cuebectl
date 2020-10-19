package cuebe

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/cuebernetes/cuebectl/pkg/ensure"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
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

	itr, err := instance.Value().Fields()
	if err != nil {
		return err
	}

	created := map[string]struct{}{}
	cont := itr.Next()
	rescan := false
	for cont {
		l := itr.Label()
		if _, ok := created[l]; ok {
			cont = itr.Next()
			continue
		}
		err := itr.Value().Validate(cue.Concrete(true))
		if err != nil {
			fmt.Printf("cannot create %s yet: %s\n", l, err)
			rescan = true
		} else {
			fmt.Printf("%s is resolved, creating\n", l)
			i, err := instanceFromCluster(itr.Value(), instance, ensurer)
			if err != nil {
				fmt.Printf("error: couldn't create object %s: %s\n", l, err)
				rescan = true
			} else {
				instance = i
				created[l] = struct{}{}
			}
		}
		cont = itr.Next()

		// restart
		if cont == false && rescan == true {
			itr, err = instance.Value().Fields()
			if err != nil {
				return err
			}
			cont = itr.Next()
			rescan = false
		}
	}
	return nil
}

// returns a new instance with the value of in from the cluster by creating in
func instanceFromCluster(in cue.Value, i *cue.Instance, ensurer ensure.Interface) (*cue.Instance, error) {
	obj := &unstructured.Unstructured{}

	if err := in.Decode(obj); err != nil {
		return nil, err
	}
	out, err := ensurer.EnsureUnstructured(obj)
	if err != nil {
		return nil, err
	}

	l, ok := in.Label()
	if !ok {
		return nil, fmt.Errorf("no label")
	}

	return i.Fill(out, l)
}
