# Files

The `files` blocks configure the file serving. Can be defined multiple times as long as the `base_path` is unique.

| Block name | Context                       | Label    | Nested block(s)           |
|:-----------|:------------------------------|:---------|:--------------------------|
| `files`    | [Server Block](server) | Optional | [CORS Block](cors) |


::attributes
---
values: [
  {
    "name": "access_control",
    "type": "tuple (string)",
    "default": "[]",
    "description": "Sets predefined access control for this block context."
  },
  {
    "name": "add_response_headers",
    "type": "object",
    "default": "",
    "description": "key/value pairs to add as response headers in the client response"
  },
  {
    "name": "base_path",
    "type": "string",
    "default": "",
    "description": "Configures the path prefix for all requests."
  },
  {
    "name": "custom_log_fields",
    "type": "object",
    "default": "",
    "description": "log fields for [custom logging](/observation/logging#custom-logging). Inherited by nested blocks."
  },
  {
    "name": "document_root",
    "type": "string",
    "default": "",
    "description": "Location of the document root (directory)."
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