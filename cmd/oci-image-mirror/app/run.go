package app

import (
	"context"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/version/verflag"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/configuration"
	"github.com/sxllwx/oci-image-mirror/pkg/ociimagemirror"
)

func init() {
	utilruntime.Must(logsapi.AddFeatureGates(utilfeature.DefaultMutableFeatureGate))
}

type Options struct {
	*configuration.Configuration
	Logs           *logs.Options
	configFilePath string
}

// NewOptions returns a new Options
func NewOptions() *Options {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	o := &Options{
		Configuration:  configuration.Default(),
		Logs:           logs.NewOptions(),
		configFilePath: path.Join(wd, "config.yaml"),
	}
	return o
}

// NewCommand provides a CLI handler for 'oci-image-mirror' command
// with a default Options.
func NewCommand(defaults *Options, ctx context.Context) *cobra.Command {
	o := *defaults
	cmd := &cobra.Command{
		Short: "Launch oci image mirror",
		Long:  "Launch oci image mirror",
		RunE: func(c *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()
			if err := logsapi.ValidateAndApply(defaults.Logs, utilfeature.DefaultFeatureGate); err != nil {
				return err
			}
			cliflag.PrintFlags(c.Flags())
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(args); err != nil {
				return err
			}
			return o.RunServer(ctx)
		},
	}

	flags := cmd.Flags()
	o.AddFlags(flags)
	return cmd
}

// Validate validates Options
func (o *Options) Validate(args []string) error {
	errors := []error{}
	return utilerrors.NewAggregate(errors)
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	logsapi.AddFlags(o.Logs, fs)
	fs.StringVarP(&o.configFilePath, "config", "c", o.configFilePath, "config file path")
}

// Complete fills in fields required to have valid data
func (o *Options) Complete() error {
	return nil
}

// Config returns config for the api server given Options
func (o *Options) Config() (*configuration.Configuration, error) {
	configFD, err := os.Open(o.configFilePath)
	if err != nil {
		return nil, err
	}
	defer configFD.Close()

	ret := o.Configuration
	return ret, yaml.NewDecoder(configFD).Decode(ret)
}

// RunServer starts a new Server given Options
func (o *Options) RunServer(ctx context.Context) error {
	config, err := o.Config()
	if err != nil {
		return err
	}
	worker := ociimagemirror.NewWorker()
	worker.Run(ctx, config)
	return nil
}
