package ociimagemirror

import (
	"strings"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/mirror"
)

type img struct {
	repository string // eg: docker.io/library/golang
	tag        string // 1.22.1
}

func newImg(image mirror.Image) img {
	return img{
		repository: image.ImageFullName(),
		tag:        image.Tag,
	}
}

func (i img) Registry() string {
	got := strings.Split(i.repository, "/")
	return got[0]
}

func (i img) Namespaces() []string {
	got := strings.Split(i.repository, "/")
	return got[1:(len(got) - 1)]
}

func (i img) Name() string {
	got := strings.Split(i.repository, "/")
	return got[(len(got) - 1)]
}

// queueItem
// We will only use structs, not pointers, because we want to use the comparison features of queues.
type queueItem struct {
	Src img
	Dst img
}
