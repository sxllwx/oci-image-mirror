package registry

import (
	"context"
	"testing"
	"time"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/mirror"
)

func TestRegistryImageExist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	testCases := []struct {
		name     string
		registry Registry
		image    mirror.Image
		expect   bool
	}{
		{
			name:     "docker hub should contain golang:1.22.1",
			registry: Registry{},
			image: mirror.Image{
				Repository: mirror.Repository{
					Registry:  "docker.io",
					Namespace: []string{"library"},
					Name:      "golang",
				},
				Tag: "1.22.1",
			},
			expect: true,
		},
		{
			name:     "docker hub should not contain golang:2.22.1",
			registry: Registry{},
			image: mirror.Image{
				Repository: mirror.Repository{
					Registry:  "docker.io",
					Namespace: []string{"library"},
					Name:      "golang",
				},
				Tag: "2.22.1",
			},
			expect: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := tc.registry.IfImageExist(ctx, tc.image)
			if err != nil {
				t.Fatal(err)
			}

			if tc.expect != ok {
				t.Fatalf("expect %v but got %v", tc.expect, ok)
			}
		})
	}
}

func TestRegistryListImageTag(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	testCases := []struct {
		name      string
		registry  Registry
		repo      mirror.Repository
		expectErr bool
	}{
		{
			name:     "docker hub should contain golang images",
			registry: Registry{},
			repo: mirror.Repository{
				Registry:  "docker.io",
				Namespace: []string{"library"},
				Name:      "golang",
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			images, err := tc.registry.ListImages(ctx, tc.repo)
			if err != nil && !tc.expectErr {
				t.Fatal(err)
			}
			for _, image := range images {
				t.Logf("%#v\n", image)
			}
		})
	}
}
