package backend

import (
	"net/http"
	"strings"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/gorilla/mux"
)

func NewEvalContext(request *http.Request, response *http.Response) *hcl.EvalContext {
	variables := make(map[string]cty.Value)

	variables["env"] = newCtyEnvMap()

	if request != nil {
		variables["req"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(request.Header),
			"params":  newCtyParametersMap(mux.Vars(request)),
		})
	}

	if response != nil {
		variables["res"] = cty.MapVal(map[string]cty.Value{
			"headers": newCtyHeadersMap(response.Header),
		})
	}

	return &hcl.EvalContext{
		Variables: variables,
		Functions: newFunctionsMap(),
	}
}


func newCtyEnvMap() cty.Value {
	ctyMap := make(map[string]cty.Value)
	for _, v := range os.Environ() {
		kv := strings.Split(v, "=") // TODO: multiple vals
		if _, ok := ctyMap[kv[0]]; !ok {
			ctyMap[kv[0]] = cty.StringVal(kv[1])
		}
	}
	return cty.MapVal(ctyMap)
}

func newCtyHeadersMap(headers http.Header) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range headers {
		ctyMap[k] = cty.StringVal(v[0]) // TODO: ListVal??
	}
	return cty.MapVal(ctyMap)
}

func newCtyParametersMap(parameters map[string]string) cty.Value {
	ctyMap := make(map[string]cty.Value)
	for k, v := range parameters {
		ctyMap[k] = cty.StringVal(v)
	}

	if len(ctyMap) == 0 {
		return cty.MapValEmpty(cty.String)
	}
	return cty.MapVal(ctyMap)
}

// Functions

func newFunctionsMap() map[string]function.Function {
	return map[string]function.Function{
		"to_upper": stdlib.UpperFunc,
		"to_lower": to_lower(), // Custom function
	}
}

// Example function
func to_lower() function.Function {
	return function.New(&function.Spec{
		Params: []function.Parameter{
			{
				Name: "s",
				Type: cty.String,
			},
		},
		Type: function.StaticReturnType(cty.String),
		Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
			s := cty.Value(args[0]).AsString()
			return cty.StringVal(strings.ToLower(s)), nil
		},
	})
}
