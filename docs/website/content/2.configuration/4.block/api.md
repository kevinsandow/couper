# API

The `api` block bundles endpoints under a certain `base_path`.

> If an error occurred for api endpoints the response gets processed
as JSON error with an error body payload. This can be customized via `error_file`.

| Block name | Context                       | Label    | Nested block(s)                                                                                                 |
|:-----------|:------------------------------|:---------|:----------------------------------------------------------------------------------------------------------------|
| `api`      | [Server Block](server) | Optional | [Endpoint Block(s)](endpoint), [CORS Block](cors), [Error Handler Block(s)](error_handler) |


### Attribute `allowed_methods`

The default value `*` can be combined with additional methods. Methods are matched case-insensitively. `Access-Control-Allow-Methods` is only sent in response to a [CORS](cors) preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method.

**Example:** `allowed_methods = ["GET", "POST"]` or `allowed_methods = ["*", "BREW"]`

::attributes
---
values: [
  {
    "name": "access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Sets predefined [access control](../access-control) for this block."
  },
  {
    "name": "add_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as response headers in the client response"
  },
  {
    "name": "allowed_methods",
    "type": "tuple (string)",
    "default": "*",
    "description": "Sets allowed methods as _default_ for all contained endpoints. Requests with a method that is not allowed result in an error response with a `405 Method Not Allowed` status."
  },
  {
    "name": "base_path",
    "type": "string",
    "default": "",
    "description": "Configures the path prefix for all requests."
  },
  {
    "name": "beta_required_permission",
    "type": "object",
    "default": "",
    "description": "Permission required to use this API (see [error type](/configuration/error-handling#error-types) `beta_insufficient_permissions`)."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "disable_access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Disables access controls by name."
  },
  {
    "name": "error_file",
    "type": "string",
    "default": "",
    "description": "Location of the error file template."
  },
  {
    "name": "remove_response_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the client response"
  },
  {
    "name": "set_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set as response headers in the client response"
  }
]

---
::