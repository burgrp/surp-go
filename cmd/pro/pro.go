package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/provider"
)

func main() {

	var r2 *provider.Register[int64]
	r2 = provider.NewIntRegister("r2", surp.NewDefined(int64(10)), true, nil, func(v surp.Optional[int64]) {
		println("r2 set:", v.String())
		r2.UpdateValue(v)
	})

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddProviders(r2)

	for {
		counter := r2.GetValue()
		if counter.IsDefined() {
			counter = surp.NewDefined(counter.Get() + 1)
		}
		r2.UpdateValue(counter)
		time.Sleep(1 * time.Second)
	}

}
