package server_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestMultiFiles_Server(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, hook := newCouperMultiFiles(
		"testdata/multi/server/couper.hcl",
		"testdata/multi/server/couper.d",
		helper,
	)
	defer shutdown()

	for _, entry := range hook.AllEntries() {
		t.Log(entry.String())
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	if err == nil || err.Error() != `Get "http://example.com:8080/": dial tcp4 127.0.0.1:8080: connect: connection refused` {
		t.Error("Expected hosts port override to 9080")
	}

	type testcase struct {
		url       string
		header    test.Header
		expStatus int
		expBody   string
	}

	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6InJzMjU2IiwidHlwIjoiSldUIn0.eyJzdWIiOjEyMzQ1Njc4OTB9.AZ0gZVqPe9TjjjJO0GnlTvERBXhPyxW_gTn050rCoEkseFRlp4TYry7WTQ7J4HNrH3btfxaEQLtTv7KooVLXQyMDujQbKU6cyuYH6MZXaM0Co3Bhu0awoX-2GVk997-7kMZx2yvwIR5ypd1CERIbNs5QcQaI4sqx_8oGrjO5ZmOWRqSpi4Mb8gJEVVccxurPu65gPFq9esVWwTf4cMQ3GGzijatnGDbRWs_igVGf8IAfmiROSVd17fShQtfthOFd19TGUswVAleOftC7-DDeJgAK8Un5xOHGRjv3ypK_6ZLRonhswaGXxovE0kLq4ZSzumQY2hOFE6x_BbrR1WKtGw"

	for _, tc := range []testcase{
		{"http://example.com:9080/", nil, http.StatusOK, "<body>RIGHT INCLUDE</body>\n"},
		{"http://example.com:9080/free", nil, http.StatusForbidden, ""},
		{"http://example.com:9080/errors/", test.Header{"Authorization": "Bearer " + token}, http.StatusTeapot, ""},
		{"http://example.com:9080/api-111", nil, http.StatusUnauthorized, ""},
		{"http://example.com:9080/api-3", nil, http.StatusTeapot, ""},
		{"http://example.com:9080/api-4/ep", nil, http.StatusNoContent, ""},
		{"http://example.com:9081/", nil, http.StatusOK, ""},
		{"http://example.com:8082/", nil, http.StatusOK, ""},
		{"http://example.com:8083/", nil, http.StatusNotFound, ""},
		{"http://example.com:9084/", nil, http.StatusNotFound, ""},
	} {
		req, err = http.NewRequest(http.MethodGet, tc.url, nil)
		helper.Must(err)

		for k, v := range tc.header {
			req.Header.Set(k, v)
		}

		res, err = client.Do(req)
		helper.Must(err)

		if res.StatusCode != tc.expStatus {
			t.Errorf("request %q: want status %d, got %d", tc.url, tc.expStatus, res.StatusCode)
		}

		if tc.expBody == "" {
			continue
		}

		var resBytes []byte
		resBytes, err = io.ReadAll(res.Body)
		helper.Must(err)
		_ = res.Body.Close()

		if string(resBytes) != tc.expBody {
			t.Errorf("request %q unexpected content: %s", tc.url, resBytes)
		}
	}
}

func TestMultiFiles_SettingsAndDefaults(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles(
		"testdata/multi/settings/couper.hcl",
		"testdata/multi/settings/couper.d",
		helper,
	)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if !strings.Contains(string(resBytes), `"Req-Id-Be-Hdr":["`) {
		t.Errorf("%s", resBytes)
	}

	if res.Header.Get("Req-Id-Cl-Hdr") == "" {
		t.Errorf("Missing 'Req-Id-Cl-Hdr' header")
	}

	// Call health route
	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/xyz", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	resBytes, err = io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); s != "healthy" {
		t.Errorf("Unexpected body given: %s", s)
	}
}

func TestMultiFiles_Definitions(t *testing.T) {
	helper := test.New(t)
	client := newClient()

	shutdown, _ := newCouperMultiFiles(
		"testdata/multi/definitions/couper.hcl",
		"testdata/multi/definitions/couper.d",
		helper,
	)
	defer shutdown()

	req, err := http.NewRequest(http.MethodGet, "http://example.com:8080/", nil)
	helper.Must(err)

	res, err := client.Do(req)
	helper.Must(err)

	resBytes, err := io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); s != "1234567890" {
		t.Errorf("Unexpected body given: %s", s)
	}

	// Call protected route
	req, err = http.NewRequest(http.MethodGet, "http://example.com:8080/added", nil)
	helper.Must(err)

	res, err = client.Do(req)
	helper.Must(err)

	resBytes, err = io.ReadAll(res.Body)
	helper.Must(err)
	_ = res.Body.Close()

	if s := string(resBytes); !strings.Contains(s, "401 access control error") {
		t.Errorf("Unexpected body given: %s", s)
	}
}