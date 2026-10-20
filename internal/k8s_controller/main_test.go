package k8s_controller

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Informers list resources with a streaming watch by default, and wait
	// for a bookmark event that the fake clientset never sends. Use the
	// classic list and watch in tests. Must be set before the first informer
	// reads the client-go feature gates.
	if err := os.Setenv("KUBE_FEATURE_WatchListClient", "false"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
