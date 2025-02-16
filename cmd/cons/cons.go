package main

import (
	"time"

	surp "github.com/burgrp/surp-go/pkg"
	"github.com/burgrp/surp-go/pkg/consumer"
)

func main() {

	// var r1 *surp.InMemoryConsumer[string]
	// r1 = surp.NewInMemoryStringConsumer("r1", func(v string) {
	// 	println(r1.Name, ":", v)
	// })

	var r2 *consumer.Register[int64]
	r2 = consumer.NewIntRegister("r2", func(value surp.Optional[int64]) {
		println(r2.GetName(), ":", value.String())
	})

	regGroup, err := surp.JoinGroup("wlp3s0", "test")
	if err != nil {
		panic(err)
	}

	defer regGroup.Close()

	regGroup.AddConsumers(r2)

	for {
		r2.SetValue(surp.NewDefined(int64(0)))
		time.Sleep(7 * time.Second)
	}
}
