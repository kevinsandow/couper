# WebSockets

The `websockets` block activates support for WebSocket connections in Couper.

| Block name   | Context                     | Label    | Nested block(s) |
|:-------------|:----------------------------|:---------|:----------------|
| `websockets` | [`proxy`](proxy)            | –        | –               |

::attributes
---
values: [
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
  },
  {
    "name": "timeout",
    "type": "string",
    "default": "",
    "description": "The total deadline [duration](#duration) a WebSocket connection has to exist."
  }
]

---
::

::duration