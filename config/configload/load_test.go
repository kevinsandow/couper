package configload_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config/configload"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/internal/test"
)

func TestPrepareBackendRefineAttributes(t *testing.T) {
	config := `server {
	endpoint "/" {
		request {
			backend "ref" {
				%s = env.VAR
			}
		}
	}
}

definitions {
	backend "ref" {
		origin = "http://localhost"
	}
}`

	for _, attribute := range []string{
		"disable_certificate_validation",
		"disable_connection_reuse",
		"http2",
		"max_connections",
	} {
		_, err := configload.LoadBytes([]byte(fmt.Sprintf(config, attribute)), "test.hcl")
		if err == nil {
			t.Fatal("expected an error")
		}

		if !strings.HasSuffix(err.Error(),
			fmt.Sprintf("backend reference: refinement for %q is not permitted; ", attribute)) {
			t.Error(err)
		}
	}
}

func TestPrepareBackendRefineBlocks(t *testing.T) {
	config := `server {
	endpoint "/" {
		request {
			backend "ref" {
				%s
			}
		}
	}
}

definitions {
	backend "ref" {
		origin = "http://localhost"
	}
}`

	_, err := configload.LoadBytes([]byte(fmt.Sprintf(config, `openapi { file = ""}`)), "test.hcl")
	if err == nil {
		t.Fatal("expected an error")
	}

	if !strings.HasSuffix(err.Error(),
		fmt.Sprintf("backend reference: refinement for %q is not permitted; ", "openapi")) {
		t.Error(err)
	}
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name  string
		hcl   string
		error string
	}{
		{
			"Bad interval",
			`interval = "10sec"`,
			`time: unknown unit "sec" in duration "10sec"`,
		},
		{
			"Bad timeout",
			`timeout = 1`,
			`time: missing unit in duration "1"`,
		},
		{
			"Bad threshold",
			`failure_threshold = -1`,
			"couper.hcl:13,29-30: Unsuitable value type; Unsuitable value: value must be a whole number, between 0 and 18446744073709551615 inclusive",
		},
		{
			"Bad expect status",
			`expected_status = [200, 204]`,
			"couper.hcl:13,27-28: Unsuitable value type; Unsuitable value: number required",
		},
		{
			"OK",
			`failure_threshold = 3
			 timeout = "3s"
			 interval = "5s"
			 expected_text = 123
			 expected_status = 200`,
			"",
		},
	}

	logger, _ := test.NewLogger()
	log := logger.WithContext(context.TODO())

	template := `
		server {
		  endpoint "/" {
		    proxy {
		      backend = "foo"
		    }
		  }
		}
		definitions {
		  backend "foo" {
		    origin = "..."
		    beta_health {
		      %%
		    }
		  }
		}`

	for _, tt := range tests {
		t.Run(tt.name, func(subT *testing.T) {
			conf, err := configload.LoadBytes([]byte(strings.Replace(template, "%%", tt.hcl, -1)), "couper.hcl")

			closeCh := make(chan struct{})
			defer close(closeCh)
			memStore := cache.New(log, closeCh)

			if conf != nil {
				_, err = runtime.NewServerConfiguration(conf, log, memStore)
			}

			var errorMsg = ""
			if err != nil {
				errorMsg = err.Error()
			}

			if tt.error != errorMsg {
				subT.Errorf("%q: Unexpected configuration error:\n\tWant: %q\n\tGot:  %q", tt.name, tt.error, errorMsg)
			}
		})
	}
}
