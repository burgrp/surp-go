package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
)

func main() {

	r1 := surp.NewInMemoryStringProvider("r1", true, nil)
	r1.SetValue("Nazdar!")

	r2 := surp.NewInMemoryIntProvider("r2", true, nil)
	r2.SetValue(-10)

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddProviders(r1, r2)

	for {
		r2.SetValue(r2.GetValue() + 1)
		time.Sleep(1 * time.Second)
	}

}
