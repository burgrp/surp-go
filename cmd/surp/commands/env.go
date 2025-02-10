package commands

import (
	"fmt"
	"os"
)

type Environment struct {
	Interface string
	Group     string
}

func GetEnvironment() (*Environment, error) {

	env := &Environment{
		Interface: os.Getenv("SURP_IF"),
		Group:     os.Getenv("SURP_GROUP"),
	}

	if env.Interface == "" {
		fmt.Println("SURP_IF environment variable is required")
		return nil, fmt.Errorf("SURP_IF environment variable is required")
	}

	if env.Group == "" {
		fmt.Println("SURP_GROUP environment variable is required")
		return nil, fmt.Errorf("SURP_GROUP environment variable is required")
	}

	return env, nil
}
