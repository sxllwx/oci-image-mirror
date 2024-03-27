package app

import (
	"fmt"
	"io"
	"net"

	"github.com/spf13/cobra"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/rest"
	netutils "k8s.io/utils/net"

	informers "github.com/sxllwx/learn-from-test/pkg/informers/externalversions"
	transportationopenapi "github.com/sxllwx/learn-from-test/pkg/openapi"
	"github.com/sxllwx/learn-from-test/pkg/webhook"
)

const defaultEtcdPathPrefix = "/registry"

// Options contains state for master/api server
type Options struct {
	RecommendedOptions *genericoptions.RecommendedOptions

	SharedInformerFactory informers.SharedInformerFactory
	StdOut                io.Writer
	StdErr                io.Writer

	AlternateDNS []string
}

// NewOptions returns a new Options
func NewOptions(out, errOut io.Writer) *Options {
	o := &Options{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			nil,
		),

		StdOut: out,
		StdErr: errOut,
	}
	o.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList = []string{"localhost:2379"}
	o.RecommendedOptions.Etcd.EnableWatchCache = false
	o.RecommendedOptions.Etcd.SkipHealthEndpoints = true
	return o
}

// NewCommand provides a CLI handler for 'start master' command
// with a default Options.
func NewCommand(defaults *Options, stopCh <-chan struct{}) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch a admission server",
		Long:  "Launch a admission server",
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			if err := o.RunServer(stopCh); err != nil {
				return err
			}
			return nil
		},
	}

	flags := cmd.Flags()
	o.RecommendedOptions.AddFlags(flags)
	utilfeature.DefaultMutableFeatureGate.AddFlag(flags)

	return cmd
}

// Validate validates Options
func (o *Options) Validate(args []string) error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

// Complete fills in fields required to have valid data
func (o *Options) Complete() error {
	// register admission plugins
	//banflunder.Register(o.RecommendedOptions.Admission.Plugins)

	// add admission plugins to the RecommendedPluginOrder
	//o.RecommendedOptions.Admission.RecommendedPluginOrder = append(o.RecommendedOptions.Admission.RecommendedPluginOrder, "BanFlunder")

	return nil
}

// Config returns config for the api server given Options
func (o *Options) Config() (*webhook.Config, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", o.AlternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(webhook.Codecs)
	serverConfig.ClientConfig = &rest.Config{}
	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(transportationopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(webhook.Scheme))
	serverConfig.OpenAPIConfig.Info.Title = "Admission"
	serverConfig.OpenAPIConfig.Info.Version = "0.1"

	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(transportationopenapi.GetOpenAPIDefinitions, openapi.NewDefinitionNamer(webhook.Scheme))
	serverConfig.OpenAPIV3Config.Info.Title = "Admission"
	serverConfig.OpenAPIV3Config.Info.Version = "0.1"

	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}

	config := &webhook.Config{
		GenericConfig: serverConfig,
		ExtraConfig:   webhook.ExtraConfig{},
	}
	return config, nil
}

// RunServer starts a new Server given Options
func (o *Options) RunServer(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New()
	if err != nil {
		return err
	}
	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}
