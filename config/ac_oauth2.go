package config

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"

	"github.com/avenga/couper/config/meta"
)

var (
	_ BackendReference      = &OAuth2AC{}
	_ BackendInitialization = &OAuth2AC{}
	_ Inline                = &OAuth2AC{}
	_ OAuth2AS              = &OAuth2AC{}
	_ OAuth2AcClient        = &OAuth2AC{}
	_ OAuth2Authorization   = &OAuth2AC{}
)

// OAuth2AC represents an oauth2 block for an OAuth2 client using the authorization code flow.
type OAuth2AC struct {
	ErrorHandlerSetter
	AuthorizationEndpoint   string   `hcl:"authorization_endpoint"` // used for lib.FnOAuthAuthorizationUrl
	BackendName             string   `hcl:"backend,optional"`
	ClientID                string   `hcl:"client_id"`
	ClientSecret            string   `hcl:"client_secret"`
	GrantType               string   `hcl:"grant_type"`
	Name                    string   `hcl:"name,label"`
	Remain                  hcl.Body `hcl:",remain"`
	Scope                   *string  `hcl:"scope,optional"`
	TokenEndpoint           string   `hcl:"token_endpoint"`
	TokenEndpointAuthMethod *string  `hcl:"token_endpoint_auth_method,optional"`
	VerifierMethod          string   `hcl:"verifier_method"`

	// internally used
	Backend hcl.Body
}

func (oa *OAuth2AC) Prepare(backendFunc PrepareBackendFunc) (err error) {
	oa.Backend, err = backendFunc("token_endpoint", oa.TokenEndpoint, oa)
	return err
}

// Reference implements the <BackendReference> interface.
func (oa *OAuth2AC) Reference() string {
	return oa.BackendName
}

// HCLBody implements the <Inline> interface.
func (oa *OAuth2AC) HCLBody() hcl.Body {
	return oa.Remain
}

// Inline implements the <Inline> interface.
func (oa *OAuth2AC) Inline() interface{} {
	type Inline struct {
		meta.LogFieldsAttribute
		Backend       *Backend `hcl:"backend,block"`
		RedirectURI   string   `hcl:"redirect_uri"`
		VerifierValue string   `hcl:"verifier_value"`
	}

	return &Inline{}
}

// Schema implements the <Inline> interface.
func (oa *OAuth2AC) Schema(inline bool) *hcl.BodySchema {
	if !inline {
		schema, _ := gohcl.ImpliedBodySchema(oa)
		return schema
	}

	schema, _ := gohcl.ImpliedBodySchema(oa.Inline())

	// A backend reference is defined, backend block is not allowed.
	if oa.BackendName != "" {
		schema.Blocks = nil
	}

	return meta.MergeSchemas(schema, meta.LogFieldsAttributeSchema)
}

func (oa *OAuth2AC) ClientAuthenticationRequired() bool {
	return true
}

func (oa *OAuth2AC) GetClientID() string {
	return oa.ClientID
}

func (oa *OAuth2AC) GetClientSecret() string {
	return oa.ClientSecret
}

func (oa *OAuth2AC) GetGrantType() string {
	return oa.GrantType
}

func (oa *OAuth2AC) GetScope() string {
	if oa.Scope == nil {
		return ""
	}
	return *oa.Scope
}

func (oa *OAuth2AC) GetAuthorizationEndpoint() (string, error) {
	return oa.AuthorizationEndpoint, nil
}

func (oa *OAuth2AC) GetTokenEndpoint() (string, error) {
	return oa.TokenEndpoint, nil
}

func (oa *OAuth2AC) GetTokenEndpointAuthMethod() *string {
	return oa.TokenEndpointAuthMethod
}

// GetVerifierMethod retrieves the verifier method (ccm_s256 or state)
func (oa *OAuth2AC) GetVerifierMethod() (string, error) {
	return oa.VerifierMethod, nil
}
