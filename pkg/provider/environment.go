package provider

import (
	"fmt"
	"os"
)

type Environment struct {
	Registry string
}

func GetEnvironment() (*Environment, error) {

	registry, ok := os.LookupEnv("SURP_REGISTRY")
	if !ok {
		return nil, fmt.Errorf("environment variable SURP_REGISTRY not set")
	}

	return &Environment{
		Registry: registry,
	}, nil
}
