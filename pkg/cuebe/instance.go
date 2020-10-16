package cuebe

import (
	"context"
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

func Do(client dynamic.Interface, mapper meta.RESTMapper, path string) error {
	var r cue.Runtime

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
			i, err := instanceFromCluster(itr.Value(), instance, client, mapper)
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

func ensure(client dynamic.Interface, mapper meta.RESTMapper, in cue.Value) (*unstructured.Unstructured, error) {
	b, err := in.MarshalJSON()
	if err != nil {
		return nil, err
	}

	obj := &unstructured.Unstructured{}
	_, gvk, err := unstructured.UnstructuredJSONScheme.Decode(b, nil, obj)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, err
	}

	var out *unstructured.Unstructured

	namespace, _ := in.Lookup("metadata", "namespace").String()

	// create if no name
	name, err := in.Lookup("metadata", "name").String()
	if err != nil || name == "" {
		out, err = client.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), obj, v1.CreateOptions{FieldManager: "cuebectl"})
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	_, err = client.Resource(mapping.Resource).Namespace(namespace).Get(context.TODO(), name, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// create if not exists
		out, err = client.Resource(mapping.Resource).Namespace(namespace).Create(context.TODO(), obj, v1.CreateOptions{FieldManager: "cuebectl"})
		if err != nil {
			return nil, err
		}
		return out, nil
	}
	if err != nil {
		// unexpected error
		return nil, err
	}

	// apply if exists
	out, err = client.Resource(mapping.Resource).Namespace(namespace).Patch(context.TODO(), name, types.ApplyPatchType, b, v1.PatchOptions{FieldManager: "cuebectl"})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// returns a new instance with the value of in from the cluster by creating in
func instanceFromCluster(in cue.Value, i *cue.Instance, client dynamic.Interface, mapper meta.RESTMapper) (*cue.Instance, error) {
	out, err := ensure(client, mapper, in)
	if err != nil {
		return nil, err
	}

	l, ok := in.Label()
	if !ok {
		return nil, fmt.Errorf("no label")
	}

	return i.Fill(out, l)
}
