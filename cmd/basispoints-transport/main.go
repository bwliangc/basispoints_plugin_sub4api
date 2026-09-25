package main

import (
	"example.com/basispoints-transport/internal/runtime"
	pluginv1 "example.com/basispoints-transport/pkg/pluginapi/v1"
)

func main() {
	pluginv1.Serve(runtime.NewServer())
}
