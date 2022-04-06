package runtime

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/sirupsen/logrus"

	"github.com/avenga/couper/cache"
	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/runtime/server"
	"github.com/avenga/couper/errors"
	"github.com/avenga/couper/eval"
	"github.com/avenga/couper/handler"
	"github.com/avenga/couper/handler/producer"
)

func newEndpointMap(srvConf *config.Server, serverOptions *server.Options) (endpointMap, error) {
	endpoints := make(endpointMap)

	apiBasePaths := make(map[string]struct{})

	for _, apiConf := range srvConf.APIs {
		basePath := serverOptions.APIBasePaths[apiConf]

		var filesBasePath, spaBasePath string
		if serverOptions.FilesBasePath != "" {
			filesBasePath = serverOptions.FilesBasePath
		}
		if serverOptions.SPABasePath != "" {
			spaBasePath = serverOptions.SPABasePath
		}

		isAPIBasePathUniqueToFilesAndSPA := basePath != filesBasePath && basePath != spaBasePath

		if _, ok := apiBasePaths[basePath]; ok {
			return nil, fmt.Errorf("API paths must be unique")
		}

		apiBasePaths[basePath] = struct{}{}

		for _, epConf := range apiConf.Endpoints {
			endpoints[epConf] = apiConf

			if epConf.Pattern == "/**" {
				isAPIBasePathUniqueToFilesAndSPA = false
			}
		}

		if isAPIBasePathUniqueToFilesAndSPA && len(newAC(srvConf, apiConf).List()) > 0 {
			endpoints[apiConf.CatchAllEndpoint] = apiConf
		}
	}

	for _, epConf := range srvConf.Endpoints {
		endpoints[epConf] = nil
	}

	return endpoints, nil
}

func newEndpointOptions(confCtx *hcl.EvalContext, endpointConf *config.Endpoint, apiConf *config.API,
	serverOptions *server.Options, log *logrus.Entry, settings *config.Settings, memStore *cache.MemoryStore) (*handler.EndpointOptions, error) {
	var errTpl *errors.Template

	if endpointConf.ErrorFile != "" {
		tpl, err := errors.NewTemplateFromFile(endpointConf.ErrorFile, log)
		if err != nil {
			return nil, err
		}
		errTpl = tpl
	} else if apiConf != nil {
		errTpl = serverOptions.APIErrTpls[apiConf]
	} else {
		errTpl = serverOptions.ServerErrTpl
	}

	// blockBodies contains inner endpoint block remain bodies to determine req/res buffer options.
	var blockBodies []hcl.Body

	var response *producer.Response
	// var redirect producer.Redirect // TODO: configure redirect block

	if endpointConf.Response != nil {
		response = &producer.Response{
			Context: endpointConf.Response.Remain,
		}
		blockBodies = append(blockBodies, response.Context)
	}

	allProxies := make(map[string]*producer.Proxy)
	for _, proxyConf := range endpointConf.Proxies {
		backend, berr := NewBackend(confCtx, proxyConf.Backend, log, settings, memStore)
		if berr != nil {
			return nil, berr
		}
		proxyHandler := handler.NewProxy(backend, proxyConf.HCLBody(), log)
		p := &producer.Proxy{
			Name:      proxyConf.Name,
			RoundTrip: proxyHandler,
		}

		allProxies[proxyConf.Name] = p
		blockBodies = append(blockBodies, proxyConf.Backend, proxyConf.HCLBody())
	}

	allRequests := make(map[string]*producer.Request)
	for _, requestConf := range endpointConf.Requests {
		backend, berr := NewBackend(confCtx, requestConf.Backend, log, settings, memStore)
		if berr != nil {
			return nil, berr
		}

		pr := &producer.Request{
			Backend: backend,
			Context: requestConf.Remain,
			Name:    requestConf.Name,
		}

		allRequests[requestConf.Name] = pr
		blockBodies = append(blockBodies, requestConf.Backend, requestConf.HCLBody())
	}

	sequences, requests, proxies := newSequences(allProxies, allRequests, endpointConf.Sequences...)

	// TODO: redirect
	if endpointConf.Response == nil && len(proxies)+len(requests)+len(sequences) == 0 { // && redirect == nil
		r := endpointConf.Remain.MissingItemRange()
		m := fmt.Sprintf("configuration error: endpoint %q requires at least one proxy, request, response or redirect block", endpointConf.Pattern)
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  m,
			Subject:  &r,
		}}
	}

	bodyLimit, err := parseBodyLimit(endpointConf.RequestBodyLimit)
	if err != nil {
		r := endpointConf.Remain.MissingItemRange()
		return nil, hcl.Diagnostics{&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "parsing endpoint request body limit: " + endpointConf.Pattern,
			Subject:  &r,
		}}
	}

	bufferOpts := eval.MustBuffer(append(blockBodies, endpointConf.Remain)...)

	apiName := ""
	if apiConf != nil {
		apiName = apiConf.Name
	}

	return &handler.EndpointOptions{
		APIName:       apiName,
		Context:       endpointConf.Remain,
		ErrorTemplate: errTpl,
		LogPattern:    endpointConf.Pattern,
		Proxies:       proxies,
		ReqBodyLimit:  bodyLimit,
		BufferOpts:    bufferOpts,
		Requests:      requests,
		Sequences:     sequences,
		Response:      response,
		ServerOpts:    serverOptions,
	}, nil
}

// newSequences lookups any request related dependency and sort them into a sequence.
// Also return left-overs for parallel usage.
func newSequences(proxies map[string]*producer.Proxy, requests map[string]*producer.Request,
	items ...*config.Sequence) (producer.Sequences, producer.Requests, producer.Proxies) {

	// just collect for filtering
	var allDeps [][]string
	for _, item := range items {
		deps := make([]string, 0)
		seen := make([]string, 0)
		resolveSequence(item, &deps, &seen)
		allDeps = append(allDeps, deps)
	}

	var reqs producer.Requests
	var ps producer.Proxies
	var seqs producer.Sequences

	// read from prepared config sequences
	for _, seq := range items {
		seqs = append(seqs, newSequence(seq, proxies, requests))
	}

proxyLeftovers:
	for name, p := range proxies {
		for _, deps := range allDeps {
			for _, dep := range deps {
				if name == dep {
					continue proxyLeftovers
				}
			}
		}
		ps = append(ps, p)
	}

reqLeftovers:
	for name, r := range requests {
		for _, deps := range allDeps {
			for _, dep := range deps {
				if name == dep {
					continue reqLeftovers
				}
			}
		}
		reqs = append(reqs, r)
	}

	return seqs, reqs, ps
}

func newSequence(seq *config.Sequence,
	proxies map[string]*producer.Proxy,
	requests map[string]*producer.Request) producer.Roundtrip {

	deps := seq.Deps()
	var rt producer.Roundtrip

	var previous []string
	if len(deps) > 1 { // more deps per item can be parallelized
		var seqs producer.Sequences
		for _, d := range deps {
			seqs = append(seqs, newSequence(d, proxies, requests))
			previous = append(previous, d.Name)
		}
		rt = seqs
	} else if len(deps) == 1 {
		rt = newSequence(deps[0], proxies, requests)
		previous = append(previous, deps[0].Name)
	}

	item := newSequenceItem(seq.Name, strings.Join(previous, ","), proxies, requests)
	if rt != nil {
		return producer.Sequence{rt, item}
	}
	return item
}

func newSequenceItem(name, previous string,
	proxies map[string]*producer.Proxy,
	requests map[string]*producer.Request) producer.Roundtrip {
	if p, ok := proxies[name]; ok {
		return producer.Proxies{
			&producer.Proxy{
				Name:             p.Name,
				RoundTrip:        p.RoundTrip,
				PreviousSequence: previous,
			}}
	}
	if r, ok := requests[name]; ok {
		return producer.Requests{
			&producer.Request{
				Backend:          r.Backend,
				Context:          r.Context,
				Name:             r.Name,
				PreviousSequence: previous,
			}}
	}
	return nil
}

func resolveSequence(item *config.Sequence, resolved, seen *[]string) {
	name := item.Name
	*seen = append(*seen, name)
	for _, dep := range item.Deps() {
		if !containsString(resolved, dep.Name) {
			if !containsString(seen, dep.Name) {
				resolveSequence(dep, resolved, seen)
				continue
			}
		}
	}

	*resolved = append(*resolved, name)
}

func containsString(slice *[]string, needle string) bool {
	for _, n := range *slice {
		if n == needle {
			return true
		}
	}
	return false
}
