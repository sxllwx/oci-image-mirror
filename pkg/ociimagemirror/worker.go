package ociimagemirror

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/configuration"
	"github.com/sxllwx/oci-image-mirror/pkg/apis/v1/mirror"
	"github.com/sxllwx/oci-image-mirror/pkg/registry"
)

type Worker struct {
	queue workqueue.RateLimitingInterface

	lock sync.Mutex
	// key is registry name(like: docker.io)
	registries map[string]*registry.Registry
}

func NewWorker() *Worker {
	return &Worker{
		queue:      workqueue.NewRateLimitingQueue(workqueue.DefaultItemBasedRateLimiter()),
		registries: map[string]*registry.Registry{},
	}
}

func (w *Worker) addRegistry(registry *registry.Registry) {
	w.lock.Lock()
	defer w.lock.Unlock()
	_, ok := w.registries[registry.Name]
	if ok {
		return
	}
	w.registries[registry.Name] = registry
}

func (w *Worker) getRegistry(name string) (*registry.Registry, bool) {
	w.lock.Lock()
	defer w.lock.Unlock()
	ret, ok := w.registries[name]
	return ret, ok
}

func (w *Worker) makeJob(ctx context.Context, config *configuration.Configuration) error {
	// register registries
	for name, r := range config.Auth {
		registry, err := registry.NewRegistry(name, registry.WithBasicAuth(r.Basic))
		if err != nil {
			return fmt.Errorf("new registry: %w", err)
		}
		w.addRegistry(registry)
	}

	eg, innerCtx := errgroup.WithContext(ctx)
	for _, src := range config.Sources {
		src := src
		eg.Go(func() error {
			registry, ok := w.getRegistry(src.Registry)
			if !ok {
				return nil
			}
			for _, name := range src.Names {
				images, err := registry.ListImages(innerCtx, mirror.Repository{
					Registry:  src.Registry,
					Namespace: src.Namespace,
					Name:      name,
				})
				if err != nil {
					return nil
				}

				for _, image := range images {
					for _, dst := range config.Destinations {
						dstImage := image
						dstImage.Registry = dst
						// just overwrite the registry name
						w.queue.Add(queueItem{
							Src: newImg(image),
							Dst: newImg(dstImage),
						})

					}
				}
			}
			return nil
		})
	}

	return eg.Wait()
}

// sync
// the core logic of mirror
// synchronize the Image in the source Registry when an Image does not exist in the target Registry.
func (w *Worker) sync(ctx context.Context, job queueItem) error {
	src, ok := w.getRegistry(job.Src.Registry())
	if !ok {
		return fmt.Errorf("registry (%s) not found", job.Src.Registry())
	}
	dst, ok := w.getRegistry(job.Dst.Registry())
	if !ok {
		return fmt.Errorf("registry (%s) not found", job.Src.Registry())
	}

	srcImageName := mirror.Image{
		Repository: mirror.Repository{
			Registry:  src.Name,
			Namespace: job.Src.Namespaces(),
			Name:      job.Src.Name(),
		},
		Tag: job.Src.tag,
	}

	dstImageName := mirror.Image{
		Repository: mirror.Repository{
			Registry:  dst.Name,
			Namespace: job.Dst.Namespaces(),
			Name:      job.Dst.Name(),
		},
		Tag: job.Src.tag,
	}

	klog.InfoS("copy image", "src", srcImageName, "dst", dstImageName)
	err := src.CopyImage(ctx, srcImageName, dstImageName, dst)
	if err != nil {
		klog.ErrorS(err, "copy image", "src", srcImageName, "dst", dstImageName)
		return err
	}
	return nil
}

func (w *Worker) worker(ctx context.Context) {
	for w.processNextWorkItem(ctx) {
	}
}

func (w *Worker) processNextWorkItem(ctx context.Context) bool {
	key, quit := w.queue.Get()
	if quit {
		return false
	}
	defer w.queue.Done(key)

	err := w.sync(ctx, key.(queueItem))
	if err == nil {
		w.queue.Forget(key)
		return true
	}

	klog.ErrorS(err, "sync image")
	w.queue.AddRateLimited(key)
	return true
}

func (w *Worker) Run(ctx context.Context, config *configuration.Configuration) {
	defer w.queue.ShutDown()

	var closeOnce sync.Once
	finished := make(chan struct{})

	go wait.UntilWithContext(ctx, func(ctx context.Context) {
		err := w.makeJob(ctx, config)
		if err != nil {
			klog.ErrorS(err, "make job")
			return
		}
		closeOnce.Do(func() {
			close(finished)
		})
		return
	}, config.Worker.Interval)

	<-finished

	for i := 0; i < int(config.Worker.Parallel); i++ {
		go wait.UntilWithContext(ctx, w.worker, time.Second)
	}
	<-ctx.Done()
}
