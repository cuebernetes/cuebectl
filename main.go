package main

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"time"
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
		l, _ := itr.Value().Label()
		if _, ok := created[l]; ok {
			// already created
			// TODO: get updates too
			cont = itr.Next()
			continue
		}
		err := itr.Value().Validate(cue.Concrete(true))
		if err != nil {
			fmt.Println("not yet concrete")
			fmt.Println(err)
			rescan = true
		} else {
			str, err := itr.Value().MarshalJSON()
			if err != nil {
				panic(err)
			}
			fmt.Println("found concrete: ")
			fmt.Println(string(str))

			// if has a name

			// concrete things get created
			kapply := exec.Command("kubectl", "create", "-o", "json", "-f", "-")
			stdin, err := kapply.StdinPipe()
			if err != nil {
				panic(err)
			}
			//kapply.Start()
			go func() {
				defer stdin.Close()
				_, err = io.WriteString(stdin, string(str))
				if err != nil {
					panic(err)
				}
			}()
			out, err := kapply.CombinedOutput()
			if err != nil {
				fmt.Println(string(out))
				panic(err)
			}
			fmt.Println(string(out))
			var un interface{}
			if err := json.Unmarshal(out, &un); err != nil {
				panic(err)
			}

			instance, err = instance.Fill(un, l)
			if err != nil {
				panic(err)
			}
			//itr.Value().Fill(un, l)
			//u, _ := itr.Value().MarshalJSON()
			//fmt.Println(string(u))
			created[l] = struct{}{}
		}
		cont = itr.Next()

		// restart
		if cont == false && rescan == true {
			time.Sleep(time.Second)
			itr, err = instance.Value().Fields()
			if err != nil {
				panic(err)
			}
			cont = itr.Next()
			rescan = false
		}

	}

	// concrete things get their values unified with the remote values
	// remaining gets checked to see if there is anything new that is concrete

}
