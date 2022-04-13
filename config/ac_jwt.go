package config

import (
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/errors"
)

var _ BackendInitialization = &JWT{}
var _ Inline = &JWT{}

// Claims represents the <Claims> object.
type Claims hcl.Expression

// JWT represents the <JWT> object.
type JWT struct {
	ErrorHandlerSetter
	BackendName           string              `hcl:"backend,optional"`
	Claims                Claims              `hcl:"claims,optional"`
	ClaimsRequired        []string            `hcl:"required_claims,optional"`
	Cookie                string              `hcl:"cookie,optional"`
	DisablePrivateCaching bool                `hcl:"disable_private_caching,optional"`
	Header                string              `hcl:"header,optional"`
	JWKsURL               string              `hcl:"jwks_url,optional"`
	JWKsTTL               string              `hcl:"jwks_ttl,optional"`
	Key                   string              `hcl:"key,optional"`
	KeyFile               string              `hcl:"key_file,optional"`
	Name                  string              `hcl:"name,label"`
	Remain                hcl.Body            `hcl:",remain"`
	RolesClaim            string              `hcl:"beta_roles_claim,optional"`
	RolesMap              map[string][]string `hcl:"beta_roles_map,optional"`
	PermissionsClaim      string              `hcl:"beta_permissions_claim,optional"`
	PermissionsMap        map[string][]string `hcl:"beta_permissions_map,optional"`
	SignatureAlgorithm    string              `hcl:"signature_algorithm,optional"`
	SigningKey            string              `hcl:"signing_key,optional"`
	SigningKeyFile        string              `hcl:"signing_key_file,optional"`
	SigningTTL            string              `hcl:"signing_ttl,optional"`
	TokenValue            hcl.Expression      `hcl:"token_value,optional"`

	// Internally used
	BodyContent *hcl.BodyContent
	Backends    map[string]hcl.Body
}

func (j *JWT) Prepare(backendFunc PrepareBackendFunc) (err error) {
	if j.JWKsURL != "" {
		if j.Backends == nil {
			j.Backends = make(map[string]hcl.Body)
		}
		j.Backends["backend"], err = backendFunc("jwks_url", j.JWKsURL, j)
		if err != nil {
			return err
		}
	}

	if err = j.check(); err != nil {
		return errors.Configuration.Label(j.Name).With(err)
	}
	return nil
}

// Reference implements the <BackendReference> interface.
func (j *JWT) Reference() string {
	return j.BackendName
}

// HCLBody implements the <Body> interface.
func (j *JWT) HCLBody() hcl.Body {
	return j.Remain
}

// Inline implements the <Inline> interface.
func (j *JWT) Inline() interface{} {
	type Inline struct {
		Backend   *Backend                  `hcl:"backend,block"`
		LogFields map[string]hcl.Expression `hcl:"custom_log_fields,optional"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (j *JWT) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(j)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(j.Inline())

	// A backend reference is defined, backend block is not allowed.
	if j.BackendName != "" {
		schema.Blocks = nil
	}

	return newBackendSchema(schema, j.HCLBody())
}

func (j *JWT) check() error {
	if j.JWKsURL == "" && j.SignatureAlgorithm == "" {
		return fmt.Errorf("signature_algorithm or jwks_url attribute required")
	}

	if j.JWKsURL == "" && (j.BackendName != "" || j.Backends != nil) {
		return fmt.Errorf("backend is obsolete without jwks_url attribute")
	} else if j.BackendName != "" && j.Backends == nil {
		return fmt.Errorf("backend must be either a block or an attribute")
	}

	if j.JWKsURL != "" {
		attributes := map[string]string{
			"signature_algorithm": j.SignatureAlgorithm,
			"key_file":            j.KeyFile,
			"key":                 j.Key,
		}

		for name, value := range attributes {
			if value != "" {
				return fmt.Errorf("%s cannot be used together with jwks_url", name)
			}
		}
	}

	return nil
}
