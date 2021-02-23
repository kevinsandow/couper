package runtime

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/pathpattern"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/sirupsen/logrus"

	ac "github.com/avenga/couper/accesscontrol"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
	"github.com/avenga/couper/handler/transport"
	"github.com/avenga/couper/handler/validation"
	"github.com/avenga/couper/internal/seetie"
	"github.com/avenga/couper/utils"
)

var DefaultBackendConf = &config.Backend{
	ConnectTimeout: "10s",
	TTFBTimeout:    "60s",
	Timeout:        "300s",
}

type Port int

func (p Port) String() string {
	return strconv.Itoa(int(p))
}

type ServerConfiguration map[Port]*MuxOptions

type hosts map[string]bool
type ports map[Port]hosts

type HandlerKind uint8

const (
	KindAPI HandlerKind = iota
	KindEndpoint
	KindFiles
	KindSPA
)

type endpointMap map[*config.Endpoint]*config.API

// NewServerConfiguration sets http handler specific defaults and validates the given gateway configuration.
// Wire up all endpoints and maps them within the returned Server.
func NewServerConfiguration(conf *config.Couper, log *logrus.Entry) (ServerConfiguration, error) {
	defaultPort := conf.Settings.DefaultPort

	// confCtx is created to evaluate request / response related configuration errors on start.
	noopReq := httptest.NewRequest(http.MethodGet, "https://couper.io", nil)
	noopResp := httptest.NewRecorder().Result()
	noopResp.Request = noopReq
	confCtx := eval.NewHTTPContext(conf.Context, 0, noopReq, noopReq, noopResp)

	validPortMap, hostsMap, err := validatePortHosts(conf, defaultPort)
	if err != nil {
		return nil, err
	}

	accessControls, err := configureAccessControls(conf, confCtx)
	if err != nil {
		return nil, err
	}

	serverConfiguration := make(ServerConfiguration)
	if len(validPortMap) == 0 {
		serverConfiguration[Port(defaultPort)] = NewMuxOptions(hostsMap)
	} else {
		for p := range validPortMap {
			serverConfiguration[p] = NewMuxOptions(hostsMap)
		}
	}

	endpointHandlers := make(map[*config.Endpoint]http.Handler)

	for _, srvConf := range conf.Servers {
		serverOptions, err := server.NewServerOptions(srvConf)
		if err != nil {
			return nil, err
		}

		var spaHandler http.Handler
		if srvConf.Spa != nil {
			spaHandler, err = handler.NewSpa(srvConf.Spa.BootstrapFile, serverOptions)
			if err != nil {
				return nil, err
			}

			spaHandler = configureProtectedHandler(accessControls, serverOptions.ServerErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Spa.AccessControl, srvConf.Spa.DisableAccessControl), spaHandler)

			for _, spaPath := range srvConf.Spa.Paths {
				err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, path.Join(serverOptions.SPABasePath, spaPath), spaHandler, KindSPA)
				if err != nil {
					return nil, err
				}
			}
		}

		if srvConf.Files != nil {
			fileHandler, err := handler.NewFile(serverOptions.FileBasePath, srvConf.Files.DocumentRoot, serverOptions)
			if err != nil {
				return nil, err
			}

			protectedFileHandler := configureProtectedHandler(accessControls, serverOptions.FileErrTpl,
				config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl),
				config.NewAccessControl(srvConf.Files.AccessControl, srvConf.Files.DisableAccessControl), fileHandler)

			err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, serverOptions.FileBasePath, protectedFileHandler, KindFiles)
			if err != nil {
				return nil, err
			}
		}

		endpointsPatterns := make(map[string]bool)

		for endpoint, parentAPI := range newEndpointMap(srvConf) {
			var basePath string
			//var cors *config.CORS
			var errTpl *errors.Template

			if parentAPI != nil {
				basePath = serverOptions.APIBasePath[parentAPI]
				//cors = parentAPI.CORS
				errTpl = serverOptions.APIErrTpl[parentAPI]
			} else {
				basePath = serverOptions.SrvBasePath
				errTpl = serverOptions.ServerErrTpl
			}

			pattern := utils.JoinPath(basePath, endpoint.Pattern)
			unique, cleanPattern := isUnique(endpointsPatterns, pattern)
			if !unique {
				return nil, fmt.Errorf("%s: duplicate endpoint: '%s'", endpoint.HCLBody().MissingItemRange().String(), pattern)
			}
			endpointsPatterns[cleanPattern] = true

			// setACHandlerFn individual wrap for access_control configuration per endpoint
			setACHandlerFn := func(protectedHandler http.Handler) {
				accessControl := config.NewAccessControl(srvConf.AccessControl, srvConf.DisableAccessControl)

				if parentAPI != nil {
					accessControl = accessControl.Merge(config.NewAccessControl(parentAPI.AccessControl, parentAPI.DisableAccessControl))
				}

				endpointHandlers[endpoint] = configureProtectedHandler(accessControls, errTpl, accessControl,
					config.NewAccessControl(endpoint.AccessControl, endpoint.DisableAccessControl),
					protectedHandler)
			}

			var proxies producer.Proxies
			var requests producer.Requests
			//var redirect producer.Redirect

			for _, proxy := range endpoint.Proxies {
				backend, berr := newBackend(confCtx, proxy.Remain, proxy.Backend, log, conf.Settings.NoProxyFromEnv)
				if berr != nil {
					return nil, berr
				}
				proxyHandler := handler.NewProxy(backend, proxy.HCLBody(), confCtx)
				proxies = append(proxies, proxyHandler)
			}

			backendConf := *DefaultBackendConf
			if diags := gohcl.DecodeBody(endpoint.Remain, confCtx, &backendConf); diags.HasErrors() {
				return nil, diags
			}
			// TODO: requests, redirect

			kind := KindEndpoint
			if parentAPI != nil {
				kind = KindAPI
			}

			bodyLimit, err := handler.ParseBodyLimit(endpoint.RequestBodyLimit)
			if err != nil {
				r := endpoint.Remain.MissingItemRange()
				return nil, hcl.Diagnostics{&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "parsing endpoint request body limit: " + endpoint.Pattern,
					Subject:  &r,
				}}
			}

			// TODO: determine req/beresp.body access in this context (all including backend) or for now:
			bufferOpts := eval.MustBuffer(endpoint.Remain)
			if len(proxies)+len(requests) > 1 { // also buffer with more possible results
				bufferOpts |= eval.BufferResponse
			}

			epOpts := &handler.EndpointOptions{
				Context:       endpoint.Remain,
				ReqBufferOpts: bufferOpts,
				ReqBodyLimit:  bodyLimit,
				Error:         errTpl,
			}
			epHandler := handler.NewEndpoint(epOpts, confCtx, log, proxies, requests)
			setACHandlerFn(epHandler)

			err = setRoutesFromHosts(serverConfiguration, defaultPort, srvConf.Hosts, pattern, endpointHandlers[endpoint], kind)
			if err != nil {
				return nil, err
			}
		}
	}

	return serverConfiguration, nil
}

func newBackend(evalCtx *hcl.EvalContext, ctx hcl.Body, beConf *config.Backend, log *logrus.Entry, ignoreProxyEnv bool) (http.RoundTripper, error) {
	tc := &transport.Config{
		BackendName:            beConf.Name,
		DisableCertValidation:  beConf.DisableCertValidation,
		DisableConnectionReuse: beConf.DisableConnectionReuse,
		HTTP2:                  beConf.HTTP2,
		NoProxyFromEnv:         ignoreProxyEnv,
		Proxy:                  beConf.Proxy,
		// TODO: parse timings /w defaults
		ConnectTimeout: 0,
		MaxConnections: 0,
		TTFBTimeout:    0,
		Timeout:        0,
	}

	openAPIopts, err := validation.NewOpenAPIOptions(beConf.OpenAPI)
	if err != nil {
		return nil, err
	}

	backend := transport.NewBackend(evalCtx, ctx, tc, log, openAPIopts)
	return backend, nil
}

func splitWildcardHostPort(host string, configuredPort int) (string, Port, error) {
	if !strings.Contains(host, ":") {
		return host, Port(configuredPort), nil
	}

	ho := host
	po := configuredPort
	h, p, err := net.SplitHostPort(host)
	if err != nil {
		return "", -1, err
	}
	ho = h
	if p != "" && p != "*" {
		if !rePortCheck.MatchString(p) {
			return "", -1, fmt.Errorf("invalid port given: %s", p)
		}
		po, err = strconv.Atoi(p)
		if err != nil {
			return "", -1, err
		}
	}

	return ho, Port(po), nil
}

func configureAccessControls(conf *config.Couper, confCtx *hcl.EvalContext) (ac.Map, error) {
	accessControls := make(ac.Map)

	if conf.Definitions != nil {
		for _, ba := range conf.Definitions.BasicAuth {
			name, err := validateACName(accessControls, ba.Name, "basic_auth")
			if err != nil {
				return nil, err
			}

			basicAuth, err := ac.NewBasicAuth(name, ba.User, ba.Pass, ba.File, ba.Realm)
			if err != nil {
				return nil, err
			}

			accessControls[name] = basicAuth
		}

		for _, jwt := range conf.Definitions.JWT {
			name, err := validateACName(accessControls, jwt.Name, "jwt")
			if err != nil {
				return nil, err
			}

			var jwtSource ac.Source
			var jwtKey string
			if jwt.Cookie != "" {
				jwtSource = ac.Cookie
				jwtKey = jwt.Cookie
			} else if jwt.Header != "" {
				jwtSource = ac.Header
				jwtKey = jwt.Header
			}
			var key []byte
			if jwt.KeyFile != "" {
				p, err := filepath.Abs(jwt.KeyFile)
				if err != nil {
					return nil, err
				}
				content, err := ioutil.ReadFile(p)
				if err != nil {
					return nil, err
				}
				key = content
			} else if jwt.Key != "" {
				key = []byte(jwt.Key)
			}

			var claims ac.Claims
			if jwt.Claims != nil {
				c, diags := seetie.ExpToMap(confCtx, jwt.Claims)
				if diags.HasErrors() {
					return nil, diags
				}
				claims = c
			}
			j, err := ac.NewJWT(jwt.SignatureAlgorithm, name, claims, jwt.ClaimsRequired, jwtSource, jwtKey, key)
			if err != nil {
				return nil, fmt.Errorf("loading jwt %q definition failed: %s", name, err)
			}

			accessControls[name] = j
		}
	}

	return accessControls, nil
}

func configureProtectedHandler(m ac.Map, errTpl *errors.Template, parentAC, handlerAC config.AccessControl, h http.Handler) http.Handler {
	var acList ac.List
	for _, acName := range parentAC.
		Merge(handlerAC).List() {
		m.MustExist(acName)
		acList = append(acList, m[acName])
	}
	if len(acList) > 0 {
		return handler.NewAccessControl(h, errTpl, acList...)
	}
	return h
}

func setRoutesFromHosts(srvConf ServerConfiguration, defaultPort int, hosts []string, path string, handler http.Handler, kind HandlerKind) error {
	hostList := hosts
	if len(hostList) == 0 {
		hostList = []string{"*"}
	}

	for _, h := range hostList {
		joinedPath := utils.JoinPath("/", path)
		host, listenPort, err := splitWildcardHostPort(h, defaultPort)
		if err != nil {
			return err
		}

		if host != "*" {
			joinedPath = utils.JoinPath(
				pathpattern.PathFromHost(
					net.JoinHostPort(host, listenPort.String()), false), "/", path)
		}

		var routes map[string]http.Handler

		switch kind {
		case KindAPI:
			fallthrough
		case KindEndpoint:
			routes = srvConf[listenPort].EndpointRoutes
		case KindFiles:
			routes = srvConf[listenPort].FileRoutes
		case KindSPA:
			routes = srvConf[listenPort].SPARoutes
		default:
			return fmt.Errorf("unknown route kind")
		}

		if _, exist := routes[joinedPath]; exist {
			return fmt.Errorf("duplicate route found on port %q: %q", listenPort.String(), path)
		}
		routes[joinedPath] = handler
	}
	return nil
}

func newEndpointMap(srvConf *config.Server) endpointMap {
	endpoints := make(endpointMap)

	for _, api := range srvConf.APIs {
		for _, endpoint := range api.Endpoints {
			endpoints[endpoint] = api
		}
	}

	for _, endpoint := range srvConf.Endpoints {
		endpoints[endpoint] = nil
	}

	return endpoints
}
