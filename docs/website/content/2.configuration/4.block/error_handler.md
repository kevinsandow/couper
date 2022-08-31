# Error Handler

The `error_handler` block lets you configure the handling of errors thrown in components configured by the parent blocks.

The error handler label specifies which [error type](/configuration/error-handling#error-types) should be handled. Multiple labels are allowed. The label can be omitted to catch all relevant errors. This has the same behavior as the error type `*`, that catches all errors explicitly.

Concerning child blocks and attributes, the `error_handler` block is similar to an [Endpoint Block](endpoint).

| Block name  |Context|Label|Nested block(s)|
| :-----------| :-----------| :-----------| :-----------|
| `error_handler` | [API Block](api), [Endpoint Block](endpoint), [Basic Auth Block](basic_auth), [JWT Block](jwt), [OAuth2 AC Block (Beta)](oauth2), [OIDC Block](oidc), [SAML Block](saml) | optional | [Proxy Block(s)](proxy),  [Request Block(s)](request), [Response Block](response), [Error Handler Block(s)](error_handler) |

## Example

```hcl
basic_auth "ba" {
  # ...
  error_handler "basic_auth_credentials_missing" {
    response {
      status = 403
      json_body = {
        error = "forbidden"
      }
    }
  }
}
```

- [Error Handling for Access Controls](https://github.com/avenga/couper-examples/blob/master/error-handling-ba/README.md).

::attributes
---
values: [
  {
    "name": "add_form_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add form parameters to the upstream request body"
  },
  {
    "name": "add_query_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add query parameters to the upstream request URL"
  },
  {
    "name": "add_request_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as request headers in the upstream request"
  },
  {
    "name": "add_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as response headers in the client response"
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "error_file",
    "type": "string",
    "default": "",
    "description": "Location of the error file template."
  },
  {
    "name": "proxy",
    "type": "object",
    "default": "",
    "description": "[`proxy`](proxy) block definition."
  },
  {
    "name": "remove_form_params",
    "type": "object",
    "default": "",
    "description": "list of names to remove form parameters from the upstream request body"
  },
  {
    "name": "remove_query_params",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove query parameters from the upstream request URL"
  },
  {
    "name": "remove_request_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the upstream request"
  },
  {
    "name": "remove_response_headers",
    "type": "tuple (string)",
    "default": "[]",
    "description": "list of names to remove headers from the client response"
  },
  {
    "name": "request",
    "type": "object",
    "default": "",
    "description": "[`request`](request) block definition."
  },
  {
    "name": "response",
    "type": "object",
    "default": "",
    "description": "[`response`](response) block definition."
  },
  {
    "name": "set_form_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL"
  },
  {
    "name": "set_query_params",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set query parameters in the upstream request URL"
  },
  {
    "name": "set_request_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to set as request headers in the upstream request"
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