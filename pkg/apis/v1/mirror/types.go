package mirror

import "path"

type Image struct {
	Repository
	Tag string
}

type Repository struct {
	Registry  string
	Namespace []string // prefix...
	Name      string
}

func (r Repository) ImageFullName() string {
	return path.Join(append(
		[]string{r.Registry}, append(r.Namespace, r.Name)...,
	)...)
}
