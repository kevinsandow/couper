# CORS

The `cors` block configures the CORS (Cross-Origin Resource Sharing) behavior in Couper.

| Block name | Context                                                                                                       | Label    | Nested block(s) |
|:-----------|:--------------------------------------------------------------------------------------------------------------|:---------|:----------------|
| `cors`     | [Server Block](server), [Files Block](files), [SPA Block](spa), [API Block](api). | no label | -               |

**Note:** `Access-Control-Allow-Methods` is only sent in response to a CORS preflight request, if the method requested by `Access-Control-Request-Method` is an allowed method (see the `allowed_method` attribute for [`api`](api) or [`endpoint`](endpoint) blocks).

### Attribute `allowed_origins`

Can be either of: a string with a single specific origin, `"*"` (all origins are allowed) or an array of specific origins.

**Example:**
```hcl
allowed_origins = ["https://www.example.com", "https://www.another.host.org"]
```

::attributes
---
values: [
  {
    "name": "allow_credentials",
    "type": "bool",
    "default": "false",
    "description": "Set to `true` if the response can be shared with credentialed requests (containing `Cookie` or `Authorization` HTTP header fields)."
  },
  {
    "name": "allowed_origins",
    "type": "object",
    "default": "",
    "description": "An allowed origin or a list of allowed origins."
  },
  {
    "name": "disable",
    "type": "bool",
    "default": "false",
    "description": "Set to `true` to disable the inheritance of CORS from parent context."
  },
  {
    "name": "max_age",
    "type": "duration",
    "default": "",
    "description": "Indicates the time the information provided by the `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` response HTTP header fields."
  }
]

---
::

::duration