# Token Request (Beta)

The `beta_token_request` block in the [Backend Block](backend) context configures a request to get a token used to authorize backend requests.

| Block name            | Context                           | Label                                                                                                                                                                                                                       | Nested block(s)                                                                                                      |
|:----------------------|:----------------------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:---------------------------------------------------------------------------------------------------------------------|
| `beta_token_request`  | [Backend Block](backend)          | &#9888; A [Token Request (Beta) Block](token_request) w/o a label has an implicit label `"default"`. Only **one** [Token Request (Beta) Block](token_request) w/ label `"default"` per [Backend Block](backend) is allowed. | [Backend Block](backend) (&#9888; required, if no `backend` block reference is defined or no `url` attribute is set. |
<!-- TODO: add available http methods -->

::attributes
---
values: [
  {
    "name": "backend",
    "type": "string",
    "default": "",
    "description": "backend block reference is required if no backend block is defined"
  },
  {
    "name": "body",
    "type": "string",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: text/plain</code> header field"
  },
  {
    "name": "expected_status",
    "type": "tuple (int)",
    "default": "[]",
    "description": "If defined, the response status code will be verified against this list of status codes, If the status code is unexpected a <code>beta_backend_token_request</code> error can be handled with an <code>error_handler</code>"
  },
  {
    "name": "form_body",
    "type": "string",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: application/x-www-form-urlencoded</code> header field."
  },
  {
    "name": "headers",
    "type": "object",
    "default": "",
    "description": "sets the given request headers"
  },
  {
    "name": "json_body",
    "type": "null, bool, number, string, object, tuple",
    "default": "",
    "description": "Creates implicit default <code>Content-Type: application/json</code> header field"
  },
  {
    "name": "query_params",
    "type": "object",
    "default": "",
    "description": "sets the url query parameters"
  },
  {
    "name": "token",
    "type": "string",
    "default": "",
    "description": "The token to be stored in <code>backends.<backend_name>.tokens.<token_request_name></code>"
  },
  {
    "name": "ttl",
    "type": "string",
    "default": "",
    "description": "The time span for which the token is to be stores."
  },
  {
    "name": "url",
    "type": "string",
    "default": "",
    "description": "If defined, the host part of the URL must be the same as the <code>origin</code> attribute of the <code>backend</code> block (if defined)."
  }
]

---
::