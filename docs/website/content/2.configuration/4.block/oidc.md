# OIDC

The `oidc` block lets you configure the [`oauth2_authorization_url()` function](/configuration/functions) and an access
control for an OIDC **Authorization Code Grant Flow** redirect endpoint.
Like all [access control](../access-control) types, the `oidc` block is defined in the [`definitions` Block](definitions) and can be referenced in all configuration blocks by its required _label_.

| Block name | Context                                 | Label            | Nested block(s)                                                              |
|:-----------|:----------------------------------------|:-----------------|:-----------------------------------------------------------------------------|
| `oidc`     | [Definitions Block](definitions) | &#9888; required | [Backend Block](backend), [Error Handler Block](error_handler) |

> any `backend` attributes: Do not disable the peer certificate validation with `disable_certificate_validation = true`.

::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "",
    "description": "`backend` block reference, defined in [`definitions`](definitions). Required, if no [`backend` block](backend) or `configuration_url` is defined within."
  },
  {
    "name": "client_id",
    "type": "string",
    "default": "",
    "description": "The client identifier."
  },
  {
    "name": "client_secret",
    "type": "string",
    "default": "",
    "description": "The client password."
  },
  {
    "name": "configuration_backend",
    "type": "object",
    "default": "",
    "description": "Optional option to configure specific behaviour for a given oidc backend."
  },
  {
    "name": "configuration_max_stale",
    "type": "duration",
    "default": "\"1h\"",
    "description": "Duration a cached OpenID configuration stays valid after its TTL has passed."
  },
  {
    "name": "configuration_ttl",
    "type": "duration",
    "default": "\"1h\"",
    "description": "The duration to cache the OpenID configuration located at `configuration_url`."
  },
  {
    "name": "configuration_url",
    "type": "string",
    "default": "",
    "description": "The OpenID configuration URL."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "jwks_max_stale",
    "type": "duration",
    "default": "\"1h\"",
    "description": "Time period the cached JWK set stays valid after its TTL has passed."
  },
  {
    "name": "jwks_ttl",
    "type": "duration",
    "default": "\"1h\"",
    "description": "Time period the JWK set stays valid and may be cached."
  },
  {
    "name": "jwks_uri_backend",
    "type": "object",
    "default": "",
    "description": "Optional option to configure specific behaviour for a given oidc backend."
  },
  {
    "name": "redirect_uri",
    "type": "string",
    "default": "",
    "description": "The Couper endpoint for receiving the authorization code. Relative URL references are resolved against the origin of the current request URL. The origin can be changed with the `accept_forwarded_url`(`settings` block) attribute if Couper is running behind a proxy."
  },
  {
    "name": "scope",
    "type": "string",
    "default": "",
    "description": "A space separated list of requested scope values for the access token."
  },
  {
    "name": "token_backend",
    "type": "object",
    "default": "",
    "description": "Optional option to configure specific behaviour for a given oidc backend."
  },
  {
    "name": "token_endpoint_auth_method",
    "type": "string",
    "default": "\"client_secret_basic\"",
    "description": "Defines the method to authenticate the client at the token endpoint. If set to `client_secret_post`, the client credentials are transported in the request body. If set to `client_secret_basic`, the client credentials are transported via Basic Authentication."
  },
  {
    "name": "userinfo_backend",
    "type": "object",
    "default": "",
    "description": "Optional option to configure specific behaviour for a given oidc backend."
  },
  {
    "name": "verifier_method",
    "type": "string",
    "default": "",
    "description": "The method to verify the integrity of the authorization code flow."
  },
  {
    "name": "verifier_value",
    "type": "string",
    "default": "",
    "description": "The value of the (unhashed) verifier."
  }
]

---
::

In most cases, referencing one `backend` (backend attribute) for all the backend requests sent by the OIDC client is enough.
You should only use `configuration_backend`, `jwks_uri_backend`, `token_backend` or `userinfo_backend` if you need to configure a specific behaviour for the respective request (e.g. timeouts).

If the OpenID server supports the `code_challenge_method` `S256` the default value for `verifier_method`is `"ccm_s256"`, `"nonce"` otherwise.

The HTTP header field `Accept: application/json` is automatically added to the token request. This can be modified with [request header modifiers](../modifiers#request-header) in a [backend block](backend).


::duration