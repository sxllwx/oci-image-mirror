package configuration

import "time"

type Configuration struct {
	Auth         map[RegistryName]RegistryConfiguration `json:"auth,omitempty" yaml:"auth,omitempty"`
	Worker       WorkerConfiguration                    `json:"worker" yaml:"worker,omitempty"`
	Sources      []Repository                           `json:"sources,omitempty" yaml:"sources,omitempty"`
	Destinations []RegistryName                         `json:"destinations,omitempty" yaml:"destinations,omitempty"`
}

type Repository struct {
	Registry  RegistryName `json:"registry,omitempty"`
	Namespace []string     `json:"namespace,omitempty"`
	Name      string       `json:"name,omitempty"`
}

type WorkerConfiguration struct {
	Parallel uint32        `json:"parallel,omitempty"`
	Interval time.Duration `json:"interval"`
}

type RegistryName = string

type RegistryConfiguration struct {
	Name  RegistryName `json:"name,omitempty"`
	Basic *Basic       `json:"basic,omitempty" yaml:"basic,omitempty"`
}

type Basic struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}
