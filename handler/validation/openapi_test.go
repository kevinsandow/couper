package validation_test

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/body"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/test"
)

// TestOpenAPIValidator_ValidateRequest should not test the openapi validation functionality but must
// ensure that all required parameters (query, body) are set as required and bodies are still readable.
func TestOpenAPIValidator_ValidateRequest(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		ct := req.Header.Get("Content-Type")
		if ct != "" {
			n, err := io.Copy(ioutil.Discard, req.Body)
			helper.Must(err)
			if n == 0 {
				t.Error("Expected body content")
			}
		}
		if req.Header.Get("Content-Type") == "application/json" {
			rw.Header().Set("Content-Type", "application/json")
			_, err := rw.Write([]byte(`{"id": 123, "name": "hans"}`))
			helper.Must(err)
		}
	}))

	log, hook := logrustest.NewNullLogger()
	logger := log.WithContext(context.Background())
	beConf := &config.Backend{
		Remain: body.New(&hcl.BodyContent{Attributes: hcl.Attributes{
			"origin": &hcl.Attribute{
				Name: "origin",
				Expr: hcltest.MockExprLiteral(cty.StringVal(origin.URL)),
			},
		}}),
		OpenAPI: &config.OpenAPI{
			File: filepath.Join("testdata/backend_01_openapi.yaml"),
		},
	}
	openAPI, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	helper.Must(err)

	evalCtx := eval.NewENVContext(nil)
	backend := transport.NewBackend(evalCtx, hcl.EmptyBody(), &transport.Config{}, logger, openAPI)
	proxy := handler.NewProxy(backend, beConf.Remain, eval.NewENVContext(nil))

	tests := []struct {
		name, method, path string
		body               io.Reader
		wantBody           bool
		wantErrLog         string
	}{
		{"GET without required query", http.MethodGet, "/a?b", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with required query", http.MethodGet, "/a?b=value", nil, false, ""},
		{"GET with required path", http.MethodGet, "/a/value", nil, false, ""},
		{"GET with required path missing", http.MethodGet, "/a//", nil, false, "request validation: Parameter 'b' in query has an error: must have a value: must have a value"},
		{"GET with optional query", http.MethodGet, "/b", nil, false, ""},
		{"GET with optional path param", http.MethodGet, "/b/a", nil, false, ""},
		{"GET with required json body", http.MethodGet, "/json", strings.NewReader(`["hans", "wurst"]`), true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, tt.body)

			if tt.body != nil {
				req.Header.Set("Content-Type", "application/json")
			}

			hook.Reset()
			res, err := proxy.RoundTrip(req)
			helper.Must(err)

			if tt.wantErrLog == "" && res.StatusCode != http.StatusOK {
				t.Errorf("Expected OK, got: %s", res.Status)
			}

			if tt.wantErrLog != "" {
				var found bool
				for _, entry := range hook.Entries {
					if entry.Message == tt.wantErrLog {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error log: %q", tt.wantErrLog)
				}
			}

			if tt.wantBody {
				n, err := io.Copy(ioutil.Discard, res.Body)
				if err != nil {
					t.Error(err)
				}
				if n == 0 {
					t.Error("Expected a response body")
				}
			}

			if t.Failed() {
				for _, entry := range hook.Entries {
					t.Log(entry.String())
				}
			}
		})
	}
}
