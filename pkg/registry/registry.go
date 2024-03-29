package registry

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/configuration"
	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/mirror"
)

type Registry struct {
	Name string
	Auth authn.Authenticator
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

func NewRegistry(name string, opts ...option) (*Registry, error) {
	ret := &Registry{
		Name: name,
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

func (r *Registry) ListImages(ctx context.Context, repository mirror.Repository) ([]mirror.Image, error) {
	registry, err := name.NewRegistry(repository.Registry)
	if err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	repo := registry.Repo(append(repository.Namespace, repository.Name)...)

	tags, err := remote.List(repo, remote.WithAuth(r.Auth), remote.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("list repository: %w", err)
	}
	var ret []mirror.Image
	for _, tag := range tags {
		ret = append(ret, mirror.Image{
			Repository: repository,
			Tag:        tag,
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
