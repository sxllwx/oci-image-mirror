package registry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"k8s.io/klog/v2"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/configuration"
	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/mirror"
)

type Registry struct {
	logger logr.Logger
	Name   string
	Auth   authn.Authenticator
}

type option func(registry *Registry) error

func WithBasicAuth(basic *configuration.Basic) option {
	return func(registry *Registry) error {
		if basic == nil {
			return nil
		}
		registry.Auth = &authn.Basic{
			Username: basic.User,
			Password: basic.Pass,
		}
		return nil
	}
}

func WithLogger(logger logr.Logger) option {
	return func(registry *Registry) error {
		registry.logger = logger
		return nil
	}
}

func NewRegistry(name string, opts ...option) (*Registry, error) {
	ret := &Registry{
		logger: klog.Background(),
		Name:   name,
	}
	for _, opt := range opts {
		err := opt(ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (r *Registry) IfImageExist(ctx context.Context, image mirror.Image) (bool, error) {
	registry, err := name.NewRegistry(image.Registry)
	if err != nil {
		return false, fmt.Errorf("parse registry: %w", err)
	}
	repository := registry.Repo(append(image.Namespace, image.Name)...)

	_, err = remote.Image(repository.Tag(image.Tag),
		remote.WithContext(ctx),
		remote.WithAuth(r.Auth),
	)
	if err != nil {
		var innerErr *transport.Error
		ok := errors.As(err, &innerErr)
		if !ok {
			return false, err
		}

		if innerErr.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, innerErr
	}
	return true, nil
}

type listOptions struct {
	limit int
}

type ListOption func(options *listOptions)

func WithLimit(limit int) func(o *listOptions) {
	return func(o *listOptions) {
		o.limit = limit
	}
}

func (r *Registry) ListImages(ctx context.Context, repository mirror.Repository, opts ...ListOption) ([]mirror.Image, error) {
	listOption := &listOptions{
		limit: math.MaxInt,
	}
	for _, opt := range opts {
		opt(listOption)
	}

	registry, err := name.NewRegistry(repository.Registry)
	if err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	repo := registry.Repo(append(repository.Namespace, repository.Name)...)

	tags, err := remote.List(repo, remote.WithAuth(r.Auth), remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list repository: %w", err)
	}

	var images []struct {
		tag        string
		createTime time.Time
	}
	for _, tag := range tags {
		// read image manifest
		image, err := remote.Image(repo.Tag(tag), remote.WithAuth(r.Auth), remote.WithContext(ctx))
		if err != nil {
			r.logger.Error(err, "inspect image", "image", repo.Tag(tag))
			continue
		}

		config, err := image.ConfigFile()
		if err != nil {
			r.logger.Error(err, "config file", "image", repo.Tag(tag))
			continue
		}
		images = append(images, struct {
			tag        string
			createTime time.Time
		}{
			tag:        tag,
			createTime: config.Created.Time,
		})
	}

	// sort by create time
	sort.Slice(images, func(i, j int) bool {
		return images[i].createTime.Before(images[j].createTime)
	})

	if len(images) > listOption.limit {
		images = images[:listOption.limit]
	}

	var ret []mirror.Image
	for _, image := range images {
		ret = append(ret, mirror.Image{
			Repository: repository,
			Tag:        image.tag,
		})
	}
	return ret, nil
}

// Maybe we will re tag the image
func (r *Registry) CopyImage(ctx context.Context, srcImage mirror.Image, dstImage mirror.Image, dst *Registry) error {
	srcRegistry, err := name.NewRegistry(srcImage.Registry)
	if err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}
	srcRepo := srcRegistry.Repo(append(srcImage.Namespace, srcImage.Name)...)

	image, err := remote.Image(srcRepo.Tag(srcImage.Tag), remote.WithAuth(r.Auth))
	if err != nil {
		return fmt.Errorf("get image: %w", err)
	}

	// if image already exist in destination repo
	ok, err := dst.IfImageExist(ctx, dstImage)
	if err != nil {
		return fmt.Errorf("image exist: %w", err)
	}
	if ok {
		return nil
	}

	registry, err := name.NewRegistry(dst.Name)
	if err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}
	repo := registry.Repo(append(srcImage.Namespace, srcImage.Name)...)

	return remote.Put(
		repo.Tag(srcImage.Tag), image,
		remote.WithContext(ctx), remote.WithAuth(dst.Auth),
	)
}
