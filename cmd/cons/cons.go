package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
)

func main() {

	// var r1 *surp.InMemoryConsumer[string]
	// r1 = surp.NewInMemoryStringConsumer("r1", func(v string) {
	// 	println(r1.Name, ":", v)
	// })

	var r2 *surp.InMemoryConsumer[int]
	r2 = surp.NewInMemoryIntConsumer("r2", func(v int) {
		println(r2.Name, ":", v)
	})

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddConsumers(r2)

	time.Sleep(10000 * time.Second)
}
