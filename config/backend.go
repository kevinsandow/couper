package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
)

var _ Inline = &Backend{}

type Backend struct {
	ConnectTimeout   string   `hcl:"connect_timeout,optional"`
	Name             string   `hcl:"name,label"`
	Options          hcl.Body `hcl:",remain"`
	RequestBodyLimit string   `hcl:"request_body_limit,optional"`
	TTFBTimeout      string   `hcl:"ttfb_timeout,optional"`
	Timeout          string   `hcl:"timeout,optional"`
}

func (b Backend) Body() hcl.Body {
	return b.Options
}

func (b Backend) Schema(inline bool) *hcl.BodySchema {
	schema, _ := gohcl.ImpliedBodySchema(b)
	if !inline {
		return schema
	}

	type Inline struct {
		Origin          string            `hcl:"origin,optional"`
		Hostname        string            `hcl:"hostname,optional"`
		Path            string            `hcl:"path,optional"`
		RequestHeaders  map[string]string `hcl:"request_headers,optional"`
		ResponseHeaders map[string]string `hcl:"response_headers,optional"`
	}

	schema, _ = gohcl.ImpliedBodySchema(&Inline{})
	return schema
}

// Merge overrides the left backend configuration and returns a new instance.
func (b *Backend) Merge(other *Backend) (*Backend, []hcl.Body) {
	if b == nil || other == nil {
		return nil, nil
	}

	var bodies []hcl.Body

	result := *b

	if other.Name != "" {
		result.Name = other.Name
	}

	if result.Options != nil {
		bodies = append(bodies, result.Options)
	}

	if other.Options != nil {
		bodies = append(bodies, other.Options)
		result.Options = other.Options
	}

	if other.ConnectTimeout != "" {
		result.ConnectTimeout = other.ConnectTimeout
	}

	if other.RequestBodyLimit != "" {
		result.RequestBodyLimit = other.RequestBodyLimit
	}

	if other.TTFBTimeout != "" {
		result.TTFBTimeout = other.TTFBTimeout
	}

	if other.Timeout != "" {
		result.Timeout = other.Timeout
	}

	return &result, bodies
}

func newBackendSchema(schema *hcl.BodySchema, body hcl.Body) *hcl.BodySchema {
	for i, block := range schema.Blocks {
		// inline backend block MAY have no label
		if block.Type == "backend" && len(block.LabelNames) > 0 {
			// check if a backend block could be parsed with label, otherwise its an inline one without label.
			content, _, _ := body.PartialContent(schema)
			if content == nil || len(content.Blocks) == 0 {
				schema.Blocks[i].LabelNames = nil
			}
		}
	}
	return schema
}
