package main

import (
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/consumer"
)

func main() {

	// var r1 *surp.InMemoryConsumer[string]
	// r1 = surp.NewInMemoryStringConsumer("r1", func(v string) {
	// 	println(r1.Name, ":", v)
	// })

	var r2 *consumer.Register[int]
	r2 = consumer.NewIntRegister("r2", func(value surp.Optional[int]) {
		println(r2.GetName(), ":", value.String())
	})

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddConsumers(r2)

	time.Sleep(10000 * time.Second)
}
