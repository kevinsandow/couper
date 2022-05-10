package server_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/zclconf/go-cty/cty"

	"github.com/avenga/couper/accesscontrol/jwt"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/eval/lib"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/internal/test"
)

func TestAccessControl_ErrorHandler(t *testing.T) {
	client := newClient()

	shutdown, logHook := newCouper("testdata/integration/error_handler/01_couper.hcl", test.New(t))
	defer shutdown()

	type testCase struct {
		name          string
		header        test.Header
		expLogMsg     string
		expStatusCode int
	}

	for _, tc := range []testCase{
		{"catch all", test.Header{"Authorization": "Basic aGFuczpoYW5z"}, "access control error: ba: credential mismatch", http.StatusNotFound},
		{"catch specific", nil, "access control error: ba: credentials required", http.StatusBadGateway},
	} {
		t.Run(tc.name, func(subT *testing.T) {
			helper := test.New(subT)

			req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
			helper.Must(err)

			tc.header.Set(req)

			res, err := client.Do(req)
			helper.Must(err)

			helper.Must(res.Body.Close())

			if res.StatusCode != tc.expStatusCode {
				subT.Fatalf("%q: expected Status %d, got: %d", tc.name, tc.expStatusCode, res.StatusCode)
			}

			if logHook.LastEntry().Data["status"] != tc.expStatusCode {
				subT.Logf("%v", logHook.LastEntry())
				subT.Errorf("Expected statusCode log: %d", tc.expStatusCode)
			}

			if logHook.LastEntry().Message != tc.expLogMsg {
				subT.Logf("%v", logHook.LastEntry())
				subT.Errorf("Expected message log: %s", tc.expLogMsg)
			}
		})
	}
}

func TestAccessControl_ErrorHandler_BasicAuth_Default(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/01_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/default/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected Status %d, got: %d", http.StatusUnauthorized, res.StatusCode)
		return
	}

	if www := res.Header.Get("www-authenticate"); www != "Basic realm=protected" {
		t.Errorf("Expected header: www-authenticate with value: %s, got: %s", "Basic realm=protected", www)
	}
}

func TestAccessControl_ErrorHandler_BasicAuth_Wildcard(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/02_couper.hcl", helper)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected Status %d, got: %d", http.StatusOK, res.StatusCode)
		return
	}

	if www := res.Header.Get("www-authenticate"); www != "" {
		t.Errorf("Expected no www-authenticate header: %s", www)
	}
}

func TestAccessControl_ErrorHandler_Configuration_Error(t *testing.T) {
	_, err := configload.LoadFiles("testdata/integration/error_handler/03_couper.hcl", "")

	expectedMsg := "03_couper.hcl:24,12-12: Missing required argument; The argument \"grant_type\" is required, but no definition was found."

	if err == nil {
		t.Error("config error should not be nil")
	} else if !strings.HasSuffix(err.Error(), expectedMsg) {
		t.Errorf("\nwant:\t%s\ngot:\t%v", expectedMsg, err.Error())
	}
}

func TestErrorHandler_Backend(t *testing.T) {
	client := test.NewHTTPClient()

	shutdown, _ := newCouper("testdata/integration/error_handler/05_couper.hcl", test.New(t))
	defer shutdown()

	type testcase struct {
		path      string
		expBody   string
		expStatus int
	}

	for _, tc := range []testcase{
		{"/api-backend", `{"api":"backend"}`, 405},
		{"/api-backend-timeout", `{"api":"backend-timeout"}`, 405},
		{"/api-backend-validation", `{"api":"backend-validation"}`, 405},
		{"/api-anything", `{"api":"backend-backend-validation"}`, 405},
		{"/backend", `{"endpoint":"backend"}`, 405},
		{"/backend-timeout", `{"endpoint":"backend-timeout"}`, 405},
		{"/backend-validation", `{"endpoint":"backend-validation"}`, 405},
		{"/anything", `{"endpoint":"backend-backend-validation"}`, 405},
		{"/c", `endpoint:backend_openapi_validation`, 405},
		{"/d", `endpoint:backend`, 405},
		{"/e", `endpoint:backend_openapi_validation`, 405},
		{"/f", `endpoint:*`, 405},
		{"/g", `endpoint:backend`, 405},
		{"/h", `endpoint:*`, 405},
	} {
		t.Run(tc.path, func(st *testing.T) {
			helper := test.New(st)

			req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Fatalf("want: %d, got: %d", tc.expStatus, res.StatusCode)
			}

			resBytes, err := io.ReadAll(res.Body)
			defer res.Body.Close()
			helper.Must(err)

			if !bytes.Contains(resBytes, []byte(tc.expBody)) {
				st.Fatalf("want: %s, got: %s", tc.expBody, resBytes)
			}
		})
	}
}

func Test_StoreInvalidBackendResponse(t *testing.T) {
	client := test.NewHTTPClient()

	shutdown, hook := newCouper("testdata/integration/error_handler/06_couper.hcl", test.New(t))
	defer shutdown()

	type testcase struct {
		path             string
		expBody          string
		expStatus        int
		expBackendStatus int
		expValidation    string
	}

	for _, tc := range []testcase{
		{"/anything", `{"req_path":"/anything","resp_ct":"application/json","resp_json_body_query":{},"resp_status":200}`, 418, 200, "status is not supported"},
	} {
		t.Run(tc.path, func(st *testing.T) {
			helper := test.New(st)
			hook.Reset()

			req, err := http.NewRequest(http.MethodGet, "http://anyserver:8080"+tc.path, nil)
			helper.Must(err)

			res, err := client.Do(req)
			helper.Must(err)

			if res.StatusCode != tc.expStatus {
				st.Errorf("status code want: %d, got: %d", tc.expStatus, res.StatusCode)
			}

			resBytes, err := io.ReadAll(res.Body)
			defer res.Body.Close()
			helper.Must(err)

			if !bytes.Contains(resBytes, []byte(tc.expBody)) {
				st.Errorf("body\nwant: %s,\ngot:  %s", tc.expBody, resBytes)
			}

			backendStatus, validation := getBackendLogStatusAndValidation(hook)
			if backendStatus != tc.expBackendStatus {
				st.Errorf("backend status want: %d, got: %d", tc.expBackendStatus, backendStatus)
			}
			if validation != tc.expValidation {
				st.Errorf("validation want: %s, got: %s", tc.expValidation, validation)
			}
		})
	}
}

func getBackendLogStatusAndValidation(hook *logrustest.Hook) (int, string) {
	for _, entry := range hook.AllEntries() {
		if entry.Data["type"] == "couper_backend" {
			return entry.Data["status"].(int), entry.Data["validation"].([]string)[0]
		}
	}

	return -1, ""
}

func TestAccessControl_ErrorHandler_Permissions(t *testing.T) {
	client := test.NewHTTPClient()

	helper := test.New(t)

	shutdown, _ := newCouper("testdata/integration/error_handler/04_couper.hcl", helper)
	defer shutdown()

	type testcase struct {
		Name               string
		Method             string
		Path               string
		GrantedPermissions []string
		ExpStatus          int
		ExpBody            string
	}

	for _, tc := range []testcase{
		{"api: sufficient permissions", http.MethodGet, "/api/", []string{"read"}, http.StatusNoContent, ""},
		{"api: insufficient permissions; handle insufficient_permission", http.MethodGet, "/api/", []string{"another"}, http.StatusTeapot, ""},
		{"api pow: sufficient permissions for method", http.MethodPost, "/api/pow/", []string{"read", "power"}, http.StatusNoContent, ""},
		{"api pow: insufficient permissions; handle insufficient_permission", http.MethodPost, "/api/pow/", []string{"read", "another"}, http.StatusBadRequest, ""},
		{"api pow: method not allowed", http.MethodGet, "/api/pow/", []string{"read", "another"}, http.StatusMethodNotAllowed, ""},
		{"endpoint: sufficient permissions", http.MethodGet, "/", []string{"write"}, http.StatusOK, ""},
		{"endpoint: insufficient permissions; handle insufficient_permission", http.MethodGet, "/", []string{"another"}, http.StatusTeapot, ""},
		{"api specific, endpoint *: insufficient permissions; handle insufficient_permission", http.MethodGet, "/wildcard1/", []string{"another"}, http.StatusBadRequest, ""},
		{"api *, endpoint specific: insufficient permissions; handle insufficient_permission", http.MethodGet, "/wildcard2/", []string{"another"}, http.StatusBadRequest, ""},
	} {
		t.Run(tc.Name, func(st *testing.T) {
			h := test.New(st)
			req, err := http.NewRequest(tc.Method, "http://localhost:8080"+tc.Path, nil)
			h.Must(err)

			conf, err := lib.NewJWTSigningConfigFromJWT(&config.JWT{
				Name:               "test",
				Key:                "s3cr3t", // same as config file
				SignatureAlgorithm: jwt.AlgorithmHMAC256.String(),
				SigningTTL:         "5m", // required for jwt sign func
			})
			h.Must(err)

			ctx := eval.ContextFromRequest(req)
			ctx = ctx.WithJWTSigningConfigs(map[string]*lib.JWTSigningConfig{
				"test": conf,
			})
			ctx = ctx.WithClientRequest(req)

			tokenVal, err := ctx.HCLContext().Functions[lib.FnJWTSign].
				Call([]cty.Value{cty.StringVal("test"), seetie.MapToValue(
					map[string]interface{}{
						"scope": tc.GrantedPermissions,
					})})
			h.Must(err)

			req.Header.Set("Authorization", "Bearer "+seetie.ValueToString(tokenVal))

			res, err := client.Do(req)
			h.Must(err)

			if res.StatusCode != tc.ExpStatus {
				st.Errorf("Expected statusCode: %d, got: %d", tc.ExpStatus, res.StatusCode)
			}
		})
	}

}

func Test_Panic_Multi_EH(t *testing.T) {
	_, err := configload.LoadFiles("testdata/settings/16_couper.hcl", "")

	expectedMsg := `: duplicate error type registration: "*"; `

	if err == nil {
		t.Error("config error should not be nil")
	} else if !strings.HasSuffix(err.Error(), expectedMsg) {
		t.Errorf("\nwant:\t'%s'\nin:\t\t'%s'", expectedMsg, err.Error())
	}
}
