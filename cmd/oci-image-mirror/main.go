package main

import (
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"

	"github.com/sxllwx/oci-image-mirror/cmd/oci-image-mirror/app"
)

func main() {
	options := app.NewOptions(os.Stdout, os.Stderr)
	cmd := app.NewCommand(options, genericapiserver.SetupSignalHandler())
	code := cli.Run(cmd)
	os.Exit(code)
}
