package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/legacy/pathpattern"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/request"
	"github.com/avenga/couper/config/runtime"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/utils"
)

// Mux is a http request router and dispatches requests
// to their corresponding http handlers.
type Mux struct {
	endpointRoot *pathpattern.Node
	fileRoot     *pathpattern.Node
	handler      map[*routers.Route]http.Handler
	opts         *runtime.MuxOptions
	spaRoot      *pathpattern.Node
}

var allowedMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
}

var fileMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
}

const (
	serverOptionsKey    = "serverContextOptions"
	wildcardReplacement = "/{_couper_wildcardMatch*}"
	wildcardSearch      = "/**"
)

func NewMux(options *runtime.MuxOptions) *Mux {
	opts := options
	if opts == nil {
		opts = runtime.NewMuxOptions()
	}

	mux := &Mux{
		opts:         opts,
		endpointRoot: &pathpattern.Node{},
		fileRoot:     &pathpattern.Node{},
		spaRoot:      &pathpattern.Node{},
		handler:      make(map[*routers.Route]http.Handler),
	}

	for path, h := range opts.EndpointRoutes {
		// TODO: handle method option per endpoint configuration
		mux.mustAddRoute(mux.endpointRoot, allowedMethods, path, h, true)
	}

	for path, h := range opts.FileRoutes {
		mux.mustAddRoute(mux.fileRoot, fileMethods, utils.JoinPath(path, "/**"), h, false)
	}

	for path, h := range opts.SPARoutes {
		mux.mustAddRoute(mux.spaRoot, fileMethods, path, h, false)
	}

	return mux
}

func (m *Mux) MustAddRoute(method, path string, handler http.Handler) *Mux {
	methods := allowedMethods[:]
	if method != "*" {
		um := strings.ToUpper(method)
		var allowed bool
		for _, am := range allowedMethods {
			if um == am {
				allowed = true
				break
			}
		}
		if !allowed {
			panic(fmt.Errorf("method not allowed: %q, path: %q", um, path))
		}

		methods = []string{um}
	}
	return m.mustAddRoute(m.endpointRoot, methods, path, handler, false)
}

func (m *Mux) mustAddRoute(root *pathpattern.Node, methods []string, path string, handler http.Handler, forEndpoint bool) *Mux {
	if forEndpoint && strings.HasSuffix(path, wildcardSearch) {
		route := mustCreateNode(root, handler, "", path)
		m.handler[route] = handler
		return m
	}

	for _, method := range methods {
		route := mustCreateNode(root, handler, method, path)
		m.handler[route] = handler
	}

	return m
}

func (m *Mux) FindHandler(req *http.Request) http.Handler {
	var route *routers.Route

	node, paramValues := m.match(m.endpointRoot, req)
	if node == nil {
		node, paramValues = m.matchWithoutMethod(m.endpointRoot, req)
	}
	if node == nil {
		// No matches for api or free endpoints. Determine if we have entered an api basePath
		// and handle api related errors accordingly.
		// Otherwise look for existing files or spa fallback.
		if tpl, api := m.getAPIErrorTemplate(req.URL.Path); tpl != nil {
			return tpl.WithError(errors.RouteNotFound.Label(api.BasePath)) // TODO: api label
		}

		fileHandler, exist := m.hasFileResponse(req)
		if exist {
			return fileHandler
		}

		node, paramValues = m.match(m.spaRoot, req)

		if node == nil {
			if fileHandler != nil {
				return fileHandler
			}

			// Fallback
			return m.opts.ServerOptions.ServerErrTpl.WithError(errors.RouteNotFound)
		}
	}

	route, _ = node.Value.(*routers.Route)

	pathParams := make(request.PathParameter, len(paramValues))
	paramKeys := node.VariableNames
	for i, value := range paramValues {
		key := paramKeys[i]
		key = strings.TrimSuffix(key, "*")
		pathParams[key] = value
	}

	ctx := req.Context()

	const wcm = "_couper_wildcardMatch"
	if wildcardMatch, ok := pathParams[wcm]; ok {
		ctx = context.WithValue(ctx, request.Wildcard, wildcardMatch)
		delete(pathParams, wcm)
	}

	ctx = context.WithValue(ctx, request.PathParams, pathParams)
	*req = *req.WithContext(ctx)

	return m.handler[route]
}

func (m *Mux) match(root *pathpattern.Node, req *http.Request) (*pathpattern.Node, []string) {
	*req = *req.WithContext(context.WithValue(req.Context(), request.ServerName, m.opts.ServerOptions.ServerName))

	return root.Match(req.Method + " " + req.URL.Path)
}

func (m *Mux) matchWithoutMethod(root *pathpattern.Node, req *http.Request) (*pathpattern.Node, []string) {
	*req = *req.WithContext(context.WithValue(req.Context(), request.ServerName, m.opts.ServerOptions.ServerName))

	return root.Match(req.URL.Path)
}

func (m *Mux) hasFileResponse(req *http.Request) (http.Handler, bool) {
	node, _ := m.match(m.fileRoot, req)
	if node == nil {
		return nil, false
	}

	route := node.Value.(*routers.Route)
	fileHandler := m.handler[route]
	unprotectedHandler := getChildHandler(fileHandler)
	if fh, ok := unprotectedHandler.(*handler.File); ok {
		return fileHandler, fh.HasResponse(req)
	}

	if fh, ok := fileHandler.(*handler.File); ok {
		return fileHandler, fh.HasResponse(req)
	}

	return fileHandler, false
}

func (m *Mux) getAPIErrorTemplate(reqPath string) (*errors.Template, *config.API) {
	for api, path := range m.opts.ServerOptions.APIBasePaths {
		if !isConfigured(path) {
			continue
		}

		if isAPIError(path, m.opts.ServerOptions.FilesBasePath, m.opts.ServerOptions.SPABasePath, reqPath) {
			return m.opts.ServerOptions.APIErrTpls[api], api
		}
	}

	return nil, nil
}

func mustCreateNode(root *pathpattern.Node, handler http.Handler, method, path string) *routers.Route {
	pathOptions := &pathpattern.Options{}

	if strings.HasSuffix(path, wildcardSearch) {
		pathOptions.SupportWildcard = true
		path = path[:len(path)-len(wildcardSearch)] + wildcardReplacement
	}

	nodePath := path
	if method != "" {
		nodePath = method + " " + path
	}

	node, err := root.CreateNode(nodePath, pathOptions)
	if err != nil {
		panic(fmt.Errorf("create path node failed: %s %q: %v", method, path, err))
	}

	var serverOpts *server.Options
	if optsHandler, ok := handler.(server.Context); ok {
		serverOpts = optsHandler.Options()
	}

	node.Value = &routers.Route{
		Method: method,
		Path:   path,
		Server: &openapi3.Server{Variables: map[string]*openapi3.ServerVariable{
			serverOptionsKey: {Default: fmt.Sprintf("%#v", serverOpts)},
		}},
	}

	return node.Value.(*routers.Route)
}

// isAPIError checks the path w/ and w/o the
// trailing slash against the request path.
func isAPIError(apiPath, filesBasePath, spaBasePath, reqPath string) bool {
	if matchesPath(apiPath, reqPath) {
		if isConfigured(filesBasePath) && apiPath == filesBasePath {
			return false
		}
		if isConfigured(spaBasePath) && apiPath == spaBasePath {
			return false
		}

		return true
	}

	return false
}

// matchesPath checks the path w/ and w/o the
// trailing slash against the request path.
func matchesPath(path, reqPath string) bool {
	p1 := path
	p2 := path

	if p1 != "/" && !strings.HasSuffix(p1, "/") {
		p1 += "/"
	}
	if p2 != "/" && strings.HasSuffix(p2, "/") {
		p2 = p2[:len(p2)-len("/")]
	}

	if strings.HasPrefix(reqPath, p1) || reqPath == p2 {
		return true
	}

	return false
}

func isConfigured(basePath string) bool {
	return basePath != ""
}
