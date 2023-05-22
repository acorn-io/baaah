package urlbuilder

import (
	"net/url"
	"path"
	"strings"
)

type PathBuilder struct {
	Prefix      string
	APIGroup    string
	APIVersion  string
	Namespace   string
	Name        string
	Resource    string
	Subresource string
	SubPath     string
}

func (u PathBuilder) URL(template *url.URL) *url.URL {
	cp := *template
	cp.Path = u.Build()
	return &cp
}

func (u PathBuilder) Build() string {
	p := u.Prefix
	if u.APIGroup != "" || u.APIVersion != "" {
		p = path.Join(p, u.APIGroup, u.APIVersion)
	}
	if u.Namespace != "" {
		p = path.Join(p, "namespaces", u.Namespace)
	}
	if u.Resource != "" {
		p = path.Join(p, strings.ToLower(u.Resource))
	}
	if u.Name != "" || u.Subresource != "" || u.SubPath != "" {
		p = path.Join(p, u.Name, u.Subresource, u.SubPath)
	}
	return p
}
