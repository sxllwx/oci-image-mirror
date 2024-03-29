package main

import (
	"os"

	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"

	"github.com/sxllwx/oci-image-mirror/cmd/oci-image-mirror/app"
)

func main() {
	options := app.NewOptions()
	cmd := app.NewCommand(options, genericapiserver.SetupSignalContext())
	code := cli.Run(cmd)
	os.Exit(code)
}
