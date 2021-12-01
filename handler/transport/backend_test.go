package transport_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	logrustest "github.com/sirupsen/logrus/hooks/test"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestBackend_RoundTrip_Timings(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodHead {
			time.Sleep(time.Second * 2) // > ttfb and overall timeout
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	tests := []struct {
		name        string
		context     hcl.Body
		tconf       *transport.Config
		req         *http.Request
		expectedErr string
	}{
		{"with zero timings", test.NewRemainContext("origin", origin.URL), &transport.Config{}, httptest.NewRequest(http.MethodGet, "http://1.2.3.4/", nil), ""},
		{"with overall timeout", test.NewRemainContext("origin", origin.URL), &transport.Config{Timeout: time.Second / 2, ConnectTimeout: time.Minute}, httptest.NewRequest(http.MethodHead, "http://1.2.3.5/", nil), "deadline exceeded"},
		{"with connect timeout", test.NewRemainContext("origin", "http://blackhole.webpagetest.org"), &transport.Config{ConnectTimeout: time.Second / 2}, httptest.NewRequest(http.MethodGet, "http://1.2.3.6/", nil), "i/o timeout"},
		{"with ttfb timeout", test.NewRemainContext("origin", origin.URL), &transport.Config{TTFBTimeout: time.Second}, httptest.NewRequest(http.MethodHead, "http://1.2.3.7/", nil), "timeout awaiting response headers"},
	}

	logger, hook := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			tt.tconf.NoProxyFromEnv = true // use origin addr from transport.Config
			backend := transport.NewBackend(tt.context, tt.tconf, nil, log)

			_, err := backend.RoundTrip(tt.req)
			if err != nil && tt.expectedErr == "" {
				subT.Error(err)
				return
			}

			gerr, isErr := err.(errors.GoError)

			if tt.expectedErr != "" &&
				(err == nil || !isErr || !strings.HasSuffix(gerr.LogError(), tt.expectedErr)) {
				subT.Errorf("Expected err %s, got: %v", tt.expectedErr, err)
			}
		})
	}
}

func TestBackend_Compression_Disabled(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Accept-Encoding") != "" {
			t.Error("Unexpected Accept-Encoding header")
		}
		rw.WriteHeader(http.StatusNoContent)
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})
	backend := transport.NewBackend(hclBody, &transport.Config{}, nil, log)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("Expected 204, got: %d", res.StatusCode)
	}
}

func TestBackend_Compression_ModifyAcceptEncoding(t *testing.T) {
	helper := test.New(t)

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if ae := req.Header.Get("Accept-Encoding"); ae != "gzip" {
			t.Errorf("Unexpected Accept-Encoding header: %s", ae)
		}

		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		for i := 1; i < 1000; i++ {
			w.Write([]byte("<html/>"))
		}
		w.Close()

		rw.Header().Set("Content-Encoding", "gzip")
		rw.Write(b.Bytes())
	}))
	defer origin.Close()

	logger, _ := logrustest.NewNullLogger()
	log := logger.WithContext(context.Background())

	u := seetie.GoToValue(origin.URL)
	hclBody := hcltest.MockBody(&hcl.BodyContent{
		Attributes: hcltest.MockAttrs(map[string]hcl.Expression{
			"origin": hcltest.MockExprLiteral(u),
		}),
	})

	backend := transport.NewBackend(hclBody, &transport.Config{
		Origin: origin.URL,
	}, nil, log)

	req := httptest.NewRequest(http.MethodOptions, "http://1.2.3.4/", nil)
	req = req.WithContext(context.WithValue(context.Background(), request.BufferOptions, eval.BufferResponse))
	req.Header.Set("Accept-Encoding", "br, gzip")
	res, err := backend.RoundTrip(req)
	helper.Must(err)

	if res.ContentLength != 60 {
		t.Errorf("Unexpected C/L: %d", res.ContentLength)
	}

	n, err := io.Copy(io.Discard, res.Body)
	helper.Must(err)

	if n != 6993 {
		t.Errorf("Unexpected body length: %d, want: %d", n, 6993)
	}
}

func TestBackend_RoundTrip_Validation(t *testing.T) {
	helper := test.New(t)
	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "text/plain")
		if req.URL.RawQuery == "404" {
			rw.WriteHeader(http.StatusNotFound)
		}
		_, err := rw.Write([]byte("from upstream"))
		helper.Must(err)
	}))
	defer origin.Close()

	openAPIYAML := helper.NewOpenAPIConf("/get")

	tests := []struct {
		name               string
		openapi            *config.OpenAPI
		requestMethod      string
		requestPath        string
		expectedErr        string
		expectedLogMessage string
	}{
		{
			"valid request / valid response",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/get",
			"",
			"",
		},
		{
			"invalid request",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodPost,
			"/get",
			"backend validation error",
			"'POST /get': method not allowed",
		},
		{
			"invalid request, IgnoreRequestViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreRequestViolations: true, IgnoreResponseViolations: true},
			http.MethodPost,
			"/get",
			"",
			"'POST /get': method not allowed",
		},
		{
			"invalid response",
			&config.OpenAPI{File: "testdata/upstream.yaml"},
			http.MethodGet,
			"/get?404",
			"backend validation error",
			"status is not supported",
		},
		{
			"invalid response, IgnoreResponseViolations",
			&config.OpenAPI{File: "testdata/upstream.yaml", IgnoreResponseViolations: true},
			http.MethodGet,
			"/get?404",
			"",
			"status is not supported",
		},
	}

	logger, hook := test.NewLogger()
	log := logger.WithContext(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hook.Reset()

			openapiValidatorOptions, err := validation.NewOpenAPIOptionsFromBytes(tt.openapi, openAPIYAML)
			if err != nil {
				subT.Fatal(err)
			}
			content := helper.NewInlineContext(`
				origin = "` + origin.URL + `"
			`)

			backend := transport.NewBackend(content, &transport.Config{}, &transport.BackendOptions{
				OpenAPI: openapiValidatorOptions,
			}, log)

			req := httptest.NewRequest(tt.requestMethod, "http://1.2.3.4"+tt.requestPath, nil)

			_, err = backend.RoundTrip(req)
			if err != nil && tt.expectedErr == "" {
				subT.Error(err)
				return
			}

			if tt.expectedErr != "" && (err == nil || err.Error() != tt.expectedErr) {
				subT.Errorf("\nwant:\t%s\ngot:\t%v", tt.expectedErr, err)
				subT.Log(hook.LastEntry().Message)
			}

			entry := hook.LastEntry()
			if tt.expectedLogMessage != "" {
				if data, ok := entry.Data["validation"]; ok {
					for _, errStr := range data.([]string) {
						if errStr != tt.expectedLogMessage {
							subT.Errorf("\nwant:\t%s\ngot:\t%v", tt.expectedLogMessage, errStr)
							return
						}
						return
					}
					for _, errStr := range data.([]string) {
						subT.Log(errStr)
					}
				}
				subT.Errorf("expected matching validation error logs:\n\t%s\n\tgot: nothing", tt.expectedLogMessage)
			}

		})
	}
}

func TestBackend_director(t *testing.T) {
	helper := test.New(t)

	log, _ := logrustest.NewNullLogger()
	nullLog := log.WithContext(context.TODO())

	bgCtx := context.Background()

	tests := []struct {
		name      string
		inlineCtx string
		path      string
		ctx       context.Context
		expReq    *http.Request
	}{
		{"proxy url settings", `origin = "http://1.2.3.4"`, "", bgCtx, httptest.NewRequest("GET", "http://1.2.3.4", nil)},
		{"proxy url settings w/hostname", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
		`, "", bgCtx, httptest.NewRequest("GET", "http://couper.io", nil)},
		{"proxy url settings w/wildcard ctx", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/**"
		`, "/peter", context.WithValue(bgCtx, request.Wildcard, "/hans"), httptest.NewRequest("GET", "http://couper.io/hans", nil)},
		{"proxy url settings w/wildcard ctx empty", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/docs/**"
		`, "", context.WithValue(bgCtx, request.Wildcard, ""), httptest.NewRequest("GET", "http://couper.io/docs", nil)},
		{"proxy url settings w/wildcard ctx empty /w trailing path slash", `
			origin = "http://1.2.3.4"
			hostname =  "couper.io"
			path = "/docs/**"
		`, "/", context.WithValue(bgCtx, request.Wildcard, ""), httptest.NewRequest("GET", "http://couper.io/docs/", nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			hclContext := helper.NewInlineContext(tt.inlineCtx)

			backend := transport.NewBackend(hclContext, &transport.Config{
				Timeout: time.Second,
			}, nil, nullLog)

			req := httptest.NewRequest(http.MethodGet, "https://example.com"+tt.path, nil)
			*req = *req.WithContext(tt.ctx)

			_, _ = backend.RoundTrip(req) // implicit director()

			attr, _ := hclContext.JustAttributes()
			hostnameExp, ok := attr["hostname"]

			if !ok && tt.expReq.Host != req.Host {
				subT.Errorf("expected same host value, want: %q, got: %q", req.Host, tt.expReq.Host)
			} else if ok {
				hostVal, _ := hostnameExp.Expr.Value(eval.NewContext(nil, nil).HCLContext())
				hostname := seetie.ValueToString(hostVal)
				if hostname != tt.expReq.Host {
					subT.Errorf("expected a configured request host: %q, got: %q", hostname, tt.expReq.Host)
				}
			}

			if req.URL.Path != tt.expReq.URL.Path {
				subT.Errorf("expected path: %q, got: %q", tt.expReq.URL.Path, req.URL.Path)
			}
		})
	}
}
