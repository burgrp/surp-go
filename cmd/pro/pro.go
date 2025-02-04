package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
)

func main() {

	r1 := surp.NewInMemoryStringProvider("r1", surp.NewValid("Nazdar!"), true, nil)

	r2 := surp.NewInMemoryIntProvider("r2", surp.NewValid(10), true, nil)

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddProviders(r1, r2)

	counter := 0

	for {
		var value surp.Optional[int]
		if counter%5 != 0 {
			value = surp.NewValid(counter)
		}
		r2.SetValue(value)
		time.Sleep(1 * time.Second)
		counter++
	}

}
