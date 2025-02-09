package surp_test

import (
	"testing"
	"time"

	surp "github.com/burgrp-go/surp/pkg"
	"github.com/burgrp-go/surp/pkg/provider"
	"github.com/stretchr/testify/require"
)

func TestHelloName(t *testing.T) {

	testReg := provider.NewStringRegister("test", surp.NewDefined("Bazar!"), true, nil, nil)

	providerGroup, err := surp.JoinGroup("wlp3s0", "test")
	require.NoError(t, err)
	require.NotNil(t, providerGroup)

	defer providerGroup.Close()

	providerGroup.AddProviders(testReg)

	time.Sleep(1 * time.Second)

}
