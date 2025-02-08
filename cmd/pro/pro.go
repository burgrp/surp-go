package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/provider"
)

func main() {

	//r1 := provider.NewStringRegister("r1", surp.NewValid("Nazdar!"), true, nil)

	var r2 *provider.Register[int]
	r2 = provider.NewIntRegister("r2", surp.NewValid(10), true, nil, func(v surp.Optional[int]) {
		println("r2 set:", v.String())
		r2.SetValue(v)
	})

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddProviders(r2)

	for {
		counter := r2.GetValue()
		if counter.IsValid() {
			counter = surp.NewValid(counter.Get() + 1)
		}
		r2.SetValue(counter)
		time.Sleep(1 * time.Second)

	}

}
