package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

func main() {
	var r cue.Runtime

	is := load.Instances([]string{"."}, &load.Config{
		Dir:        "example",
	})
	var instance *cue.Instance
	for _, i := range is {
		if instance == nil {
			i2, err := r.Build(i)
			if err != nil {
				panic(err)
			}
			instance = i2
		}
		instance.Build(i)
	}

	itr, err := instance.Value().Fields()
	if err != nil {
		panic(err)
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
			i, err := instanceFromCluster(itr.Value(), instance)
			if err != nil {
				fmt.Printf("error: couldn't create object %s\n", l)
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
				panic(err)
			}
			cont = itr.Next()
			rescan = false
		}
	}
}

func create(in string) ([]byte, error) {
	kapply := exec.Command("kubectl", "create", "-o", "json", "-f", "-")
	stdin, err := kapply.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		defer stdin.Close()
		_, err = io.WriteString(stdin, in)
		if err != nil {
			panic(err)
		}
	}()


	return kapply.CombinedOutput()
}

func apply(in string) ([]byte, error) {
	kapply := exec.Command("kubectl", "apply", "-o", "json", "-f", "-")
	stdin, err := kapply.StdinPipe()
	if err != nil {
		panic(err)
	}
	go func() {
		defer stdin.Close()
		_, err = io.WriteString(stdin, in)
		if err != nil {
			panic(err)
		}
	}()


	return kapply.CombinedOutput()
}

// returns a new instance with the value of in from the cluster by creating in
func instanceFromCluster(in cue.Value, i *cue.Instance) (*cue.Instance, error) {
	str, err := in.MarshalJSON()
	if err != nil {
		panic(err)
	}

	var out []byte
	hasName := true
	if name, err := in.Lookup("metadata", "name").String(); err != nil || name == "" {
		hasName = false
	}

	if !hasName {
		out, err = create(string(str))
		if err != nil {
			return nil, err
		}
	} else {
		out, err = apply(string(str))
		if err != nil {
			return nil, err
		}
	}

	var un interface{}
	if err := json.Unmarshal(out, &un); err != nil {
		panic(err)
	}

	l, ok := in.Label()
	if !ok {
		return nil, fmt.Errorf("no label")
	}

	return i.Fill(un, l)
}
