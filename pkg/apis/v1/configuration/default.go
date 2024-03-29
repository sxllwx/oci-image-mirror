package configuration

import "time"

func Default() *Configuration {
	return &Configuration{
		Auth: map[RegistryName]RegistryConfiguration{
			"docker.io": {
				Name:  "docker.io",
				Basic: nil,
			},
		},
		Worker: WorkerConfiguration{
			Parallel: 1,
			Interval: time.Minute,
		},
		Sources:      make([]Repository, 0),
		Destinations: make([]RegistryName, 0),
	}
}
