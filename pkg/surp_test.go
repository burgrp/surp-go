package surp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHelloName(t *testing.T) {

	testReg := NewInMemoryStringProvider("test", true, nil)
	testReg.SetValue("Bazar!")

	providerGroup, err := JoinGroup("wlp3s0", "test")
	require.NoError(t, err)
	require.NotNil(t, providerGroup)

	defer providerGroup.Close()

	providerGroup.AddProviders(testReg)

	time.Sleep(10000 * time.Second)

}
