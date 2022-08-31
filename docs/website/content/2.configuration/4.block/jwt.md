# JWT

The `jwt` block lets you configure JSON Web Token access control for your gateway.
Like all [access control](../access-control) types, the `jwt` block is defined in
the [`definitions` Block](definitions) and can be referenced in all configuration blocks by its
required _label_.

Since responses from endpoints protected by JWT access controls are not publicly cacheable, a `Cache-Control: private` header field is added to the response, unless this feature is disabled with `disable_private_caching = true`.

| Block name | Context                                 | Label            | Nested block(s)                                                                  |
|:-----------|:----------------------------------------|:-----------------|:---------------------------------------------------------------------------------|
| `jwt`      | [Definitions Block](definitions) | &#9888; required | [JWKS `backend`](backend), [Error Handler Block(s)](error_handler) |

::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "",
    "description": "[`backend` block](backend) reference for enhancing JWKS requests."
  },
  {
    "name": "beta_permissions_claim",
    "type": "string",
    "default": "",
    "description": "Name of claim containing the granted permissions. The claim value must either be a string containing a space-separated list of permissions or a list of string permissions."
  },
  {
    "name": "beta_permissions_map",
    "type": "object",
    "default": "",
    "description": "Mapping of granted permissions to additional granted permissions. Maps values from `beta_permissions_claim` and those created from `beta_roles_map`. The map is called recursively."
  },
  {
    "name": "beta_roles_claim",
    "type": "string",
    "default": "",
    "description": "Name of claim specifying the roles of the user represented by the token. The claim value must either be a string containing a space-separated list of role values or a list of string role values."
  },
  {
    "name": "beta_roles_map",
    "type": "object",
    "default": "",
    "description": "Mapping of roles to granted permissions. Non-mapped roles can be assigned with `*` to specific permissions."
  },
  {
    "name": "claims",
    "type": "object",
    "default": "",
    "description": "Object with claims that must be given for a valid token (equals comparison with JWT payload). The claim values are evaluated per request."
  },
  {
    "name": "cookie",
    "type": "string",
    "default": "",
    "description": "Read token value from a cookie. Cannot be used together with `header` or `token_value`"
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "disable_private_caching",
    "type": "bool",
    "default": "false",
    "description": "If set to `true`, Couper does not add the `private` directive to the `Cache-Control` HTTP header field value."
  },
  {
    "name": "header",
    "type": "string",
    "default": "",
    "description": "Read token value from the given request header field. Implies `Bearer` if `Authorization` (case-insensitive) is used, otherwise any other header name can be used. Cannot be used together with `cookie` or `token_value`."
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
    "name": "jwks_url",
    "type": "string",
    "default": "",
    "description": "URI pointing to a set of [JSON Web Keys (RFC 7517)](https://datatracker.ietf.org/doc/html/rfc7517)"
  },
  {
    "name": "key",
    "type": "string",
    "default": "",
    "description": "Public key (in PEM format) for `RS*` and `ES*` variants or the secret for `HS*` algorithm."
  },
  {
    "name": "key_file",
    "type": "string",
    "default": "",
    "description": "Optional file reference instead of `key` usage."
  },
  {
    "name": "required_claims",
    "type": "tuple (string)",
    "default": "[]",
    "description": "List of claim names that must be given for a valid token."
  },
  {
    "name": "signature_algorithm",
    "type": "string",
    "default": "",
    "description": "Valid values: `RS256`, `RS384`, `RS512`, `HS256`, `HS384`, `HS512`, `ES256`, `ES384`, `ES512`"
  },
  {
    "name": "signing_key",
    "type": "string",
    "default": "",
    "description": "Private key (in PEM format) for `RS*` and `ES*` variants."
  },
  {
    "name": "signing_key_file",
    "type": "string",
    "default": "",
    "description": "Optional file reference instead of `signing_key` usage."
  },
  {
    "name": "signing_ttl",
    "type": "duration",
    "default": "",
    "description": "The token's time-to-live (creates the `exp` claim)."
  },
  {
    "name": "token_value",
    "type": "object",
    "default": "",
    "description": "Expression to obtain the token. Cannot be used together with `cookie` or `header`."
  }
]

---
::

The attributes `header`, `cookie` and `token_value` are mutually exclusive.
If all three attributes are missing, `header = "Authorization"` will be implied, i.e. the token will be read from the incoming `Authorization` header.

If the key to verify the signatures of tokens does not change over time, it should be specified via either `key` or `key_file` (together with `signature_algorithm`).
Otherwise, a JSON web key set should be referenced via `jwks_url`; in this case, the tokens need a `kid` header.

A JWT access control configured by this block can extract permissions from

- the value of the claim specified by `beta_permissions_claim` and
- the result of mapping the value of the claim specified by `beta_roles_claim` using the `beta_roles_map`.

The `jwt` block may also be referenced by the [`jwt_sign()` function](../functions), if it has a `signing_ttl` defined. For `HS*` algorithms the signing key is taken from `key`/`key_file`, for `RS*` and `ES*` algorithms, `signing_key` or `signing_key_file` have to be specified.

> **Note:** A `jwt` block with `signing_ttl` cannot have the same label as a `jwt_signing_profile` block.

::duration