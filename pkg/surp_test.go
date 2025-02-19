package surp_test

import (
	"testing"
	"time"

	surp "github.com/burgrp/surp-go/pkg"
	"github.com/burgrp/surp-go/pkg/provider"
	"github.com/stretchr/testify/require"
)

func TestHelloName(t *testing.T) {

	testReg := provider.NewStringRegister("test", surp.NewDefined("Bazar!"), true, nil, nil)

	providerGroup, err := surp.JoinGroup("wlp3s0", "test", false)
	require.NoError(t, err)
	require.NotNil(t, providerGroup)

	defer providerGroup.Close()

	err = providerGroup.AddProviders(testReg)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

}
