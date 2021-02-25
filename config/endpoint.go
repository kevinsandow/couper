package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var _ Inline = &Endpoint{}

// Endpoint represents the <Endpoint> object.
type Endpoint struct {
	AccessControl        []string  `hcl:"access_control,optional"`
	DisableAccessControl []string  `hcl:"disable_access_control,optional"`
	Pattern              string    `hcl:"pattern,label"`
	Proxies              Proxies   `hcl:"proxy,block"`
	Remain               hcl.Body  `hcl:",remain"`
	RequestBodyLimit     string    `hcl:"request_body_limit,optional"`
	Requests             Requests  `hcl:"request,block"`
	Response             *Response `hcl:"response,block"`
}

// Endpoints represents a list of <Endpoint> objects.
type Endpoints []*Endpoint

// HCLBody implements the <Inline> interface.
func (e Endpoint) HCLBody() hcl.Body {
	return e.Remain
}

// Reference implements the <Inline> interface.
func (e Endpoint) Reference() string {
	return e.Pattern
}

// Schema implements the <Inline> interface.
func (e Endpoint) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(e)
		return schema
	}

	return meta.AttributesSchema
}
