//go:build exclude

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"

	"github.com/avenga/couper/config"
	"github.com/avenga/couper/config/meta"
)

type entry struct {
	Attributes  []attr `json:"attributes"`
	Description string `json:"description"`
	ID          string `json:"objectID"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
}

type attr struct {
	Default     string `json:"default"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Type        string `json:"type"`
}

const (
	searchAppID     = "MSIN2HU7WH"
	searchIndex     = "docs"
	searchClientKey = "SEARCH_CLIENT_API_KEY"

	configurationPath = "docs/website/content/2.configuration"
	docsBlockPath     = configurationPath + "/4.block"

	urlBasePath = "/configuration/"
)

// export md: 1) search for ::attribute, replace if exist or append at end
func main() {

	client := search.NewClient(searchAppID, os.Getenv(searchClientKey))
	index := client.InitIndex(searchIndex)

	filenameRegex := regexp.MustCompile(`(URL|JWT|OpenAPI|[a-z0-9]+)`)
	bracesRegex := regexp.MustCompile(`{([^}]*)}`)

	attributesMap := map[string][]reflect.StructField{
		"RequestHeadersAttributes":  newFields(&meta.RequestHeadersAttributes{}),
		"ResponseHeadersAttributes": newFields(&meta.ResponseHeadersAttributes{}),
		"FormParamsAttributes":      newFields(&meta.FormParamsAttributes{}),
		"QueryParamsAttributes":     newFields(&meta.QueryParamsAttributes{}),
		"LogFieldsAttribute":        newFields(&meta.LogFieldsAttribute{}),
	}

	blockNamesMap := map[string]string{
		"oauth2_ac":       "beta_oauth2",
		"oauth2_req_auth": "oauth2",
	}

	processedFiles := make(map[string]struct{})

	for _, impl := range []interface{}{
		&config.API{},
		&config.Backend{},
		&config.BackendTLS{},
		&config.BasicAuth{},
		&config.CORS{},
		&config.Defaults{},
		&config.Endpoint{},
		&config.ErrorHandler{},
		&config.Files{},
		&config.Health{},
		&config.JWTSigningProfile{},
		&config.JWT{},
		&config.OAuth2AC{},
		&config.OAuth2ReqAuth{},
		&config.OIDC{},
		&config.OpenAPI{},
		&config.Proxy{},
		&config.RateLimit{},
		&config.Request{},
		&config.Response{},
		&config.SAML{},
		&config.Server{},
		&config.ClientCertificate{},
		&config.ServerCertificate{},
		&config.Settings{},
		&config.Spa{},
		&config.TokenRequest{},
		&config.Websockets{},
	} {
		t := reflect.TypeOf(impl).Elem()
		name := reflect.TypeOf(impl).String()
		name = strings.TrimPrefix(name, "*config.")
		blockName := strings.ToLower(strings.Trim(filenameRegex.ReplaceAllString(name, "${1}_"), "_"))

		if _, exists := blockNamesMap[blockName]; exists {
			blockName = blockNamesMap[blockName]
		}

		urlPath, _ := url.JoinPath(urlBasePath, "block", blockName)
		result := entry{
			Name: blockName,
			URL:  strings.ToLower(urlPath),
			Type: "block",
		}

		result.ID = result.URL

		var fields []reflect.StructField
		for i := 0; i < t.NumField(); i++ {
			fields = append(fields, t.Field(i))
		}

		inlineType, ok := impl.(config.Inline)
		if ok {
			it := reflect.TypeOf(inlineType.Inline()).Elem()
			for i := 0; i < it.NumField(); i++ {
				field := it.Field(i)
				if _, ok := attributesMap[field.Name]; ok {
					fields = append(fields, attributesMap[field.Name]...)
				} else {
					fields = append(fields, field)
				}
			}
		}

		for _, field := range fields {
			if field.Tag.Get("docs") == "" {
				continue
			}

			fieldType := field.Tag.Get("type")
			if fieldType == "" {
				ft := strings.Replace(field.Type.String(), "*", "", 1)
				if ft == "config.List" {
					ft = "[]string"
				}
				if ft[:2] == "[]" {
					ft = "tuple (" + ft[2:] + ")"
				} else if strings.Contains(ft, "int") {
					ft = "number"
				} else if ft != "string" && ft != "bool" {
					ft = "object"
				}
				fieldType = ft
			}

			fieldDefault := field.Tag.Get("default")
			if fieldDefault == "" && fieldType == "bool" {
				fieldDefault = "false"
			} else if fieldDefault == "" && strings.HasPrefix(fieldType, "tuple ") {
				fieldDefault = "[]"
			} else if fieldDefault != "" && (fieldType == "string" || fieldType == "duration") {
				fieldDefault = `"` + fieldDefault + `"`
			}

			fieldDescription := field.Tag.Get("docs")
			fieldDescription = bracesRegex.ReplaceAllString(fieldDescription, "`${1}`")

			a := attr{
				Default:     fieldDefault,
				Description: fieldDescription,
				Name:        strings.Split(field.Tag.Get("hcl"), ",")[0],
				Type:        fieldType,
			}
			result.Attributes = append(result.Attributes, a)
		}

		sort.Sort(byName(result.Attributes))

		b := &bytes.Buffer{}
		enc := json.NewEncoder(b)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result.Attributes); err != nil {
			panic(err)
		}

		// TODO: write func
		file, err := os.OpenFile(filepath.Join(docsBlockPath, blockName+".md"), os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			panic(err)
		}

		fileBytes := &bytes.Buffer{}

		scanner := bufio.NewScanner(file)
		var skipMode, seen bool
		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "::attributes") {
				fileBytes.WriteString(fmt.Sprintf(`::attributes
---
values: %s
---
::
`, b.String()))
				skipMode = true
				seen = true
				continue
			}

			if skipMode && line == "::" {
				skipMode = false
				continue
			}

			if !skipMode {
				fileBytes.Write(scanner.Bytes())
				fileBytes.Write([]byte("\n"))
			}
		}

		if !seen { // TODO: from func/template
			fileBytes.WriteString(fmt.Sprintf(`
::attributes
---
values: %s
---
::
`, b.String()))
		}

		size, err := file.WriteAt(fileBytes.Bytes(), 0)
		if err != nil {
			panic(err)
		}
		err = os.Truncate(file.Name(), int64(size))
		if err != nil {
			panic(err)
		}

		processedFiles[file.Name()] = struct{}{}
		println("Attributes written: "+blockName+":\r\t\t\t\t\t", file.Name())

		if os.Getenv(searchClientKey) != "" {
			_, err = index.SaveObjects(result) //, opt.AutoGenerateObjectIDIfNotExist(true))
			if err != nil {
				panic(err)
			}
			println("SearchIndex updated")
		}
	}

	if os.Getenv(searchClientKey) == "" {
		return
	}

	// index non generated markdown
	indexDirectory(configurationPath, "", processedFiles, index)
	indexDirectory(docsBlockPath, "block", processedFiles, index)
}

var mdHeaderRegex = regexp.MustCompile(`#(.+)\n(\n(.+)\n)`)
var mdFileRegex = regexp.MustCompile(`\d?\.?(.+)\.md`)

func indexDirectory(dirPath, docType string, processedFiles map[string]struct{}, index *search.Index) {
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}

		entryPath := filepath.Join(dirPath, dirEntry.Name())
		if _, ok := processedFiles[entryPath]; ok {
			continue
		}

		println("Indexing from file: " + dirEntry.Name())
		fileContent, rerr := os.ReadFile(entryPath)
		if rerr != nil {
			panic(err)
		}
		println(dirEntry.Name())
		fileName := mdFileRegex.FindStringSubmatch(dirEntry.Name())[1]
		dt := docType
		if dt == "" {
			dt = fileName
		} else {
			fileName, _ = url.JoinPath(dt, fileName)
		}
		title, description, indexTable := headerFromMeta(fileContent)
		if title == "" && description == "" {
			matches := mdHeaderRegex.FindSubmatch(fileContent)
			description = string(bytes.ToLower(matches[3]))
			title = string(bytes.ToLower(matches[1]))
		}

		urlPath, _ := url.JoinPath(urlBasePath, fileName)
		result := &entry{
			Attributes:  attributesFromTable(fileContent, indexTable),
			Description: description,
			ID:          urlPath,
			Name:        title,
			Type:        dt,
			URL:         urlPath,
		}

		// debug
		if index == nil {
			b, merr := json.Marshal(result)
			if merr != nil {
				panic(merr)
			}
			println(string(b))
		} else {
			_, err = index.SaveObjects(result)
			if err != nil {
				panic(err)
			}
			println("SearchIndex updated")
		}
	}
}

func headerFromMeta(content []byte) (title string, description string, indexTable bool) {
	var metaSep = []byte(`---`)
	if !bytes.HasPrefix(content, metaSep) {
		return
	}
	endIdx := bytes.LastIndex(content, metaSep)
	s := bufio.NewScanner(bytes.NewReader(content[3:endIdx]))
	for s.Scan() {
		t := s.Text()
		if strings.HasPrefix(t, "title") {
			title = strings.Split(t, ": ")[1]
		} else if strings.HasPrefix(t, "description") {
			description = strings.Split(t, ": ")[1]
		} else if strings.HasPrefix(t, "indexTable") {
			indexTable = t == "indexTable: true"
		}

	}
	return
}

var tableEntryRegex = regexp.MustCompile(`^\|\s\x60(.+)\x60\s+\|\s(.+)\s\|\s(.+)\.\s+\|`)

func attributesFromTable(content []byte, parse bool) []attr {
	if !parse {
		return nil
	}
	attrs := make([]attr, 0)
	s := bufio.NewScanner(bytes.NewReader(content))
	var tableHeadSeen bool
	for s.Scan() {
		// scan to table header
		line := s.Text()
		if !tableHeadSeen {
			if strings.HasPrefix(line, "|:-") {
				tableHeadSeen = true
			}
			continue
		}
		if line[0] != '|' {
			break
		}
		matches := tableEntryRegex.FindStringSubmatch(line)
		if len(matches) < 4 {
			continue
		}
		attrs = append(attrs, attr{
			Description: strings.TrimSpace(matches[3]),
			Name:        strings.TrimSpace(matches[1]),
			Type:        strings.TrimSpace(matches[2]),
		})
	}
	sort.Sort(byName(attrs))
	return attrs
}

type byName []attr

func (attributes byName) Len() int {
	return len(attributes)
}
func (attributes byName) Swap(i, j int) {
	attributes[i], attributes[j] = attributes[j], attributes[i]
}
func (attributes byName) Less(i, j int) bool {
	return attributes[i].Name < attributes[j].Name
}

func newFields(impl interface{}) []reflect.StructField {
	it := reflect.TypeOf(impl).Elem()
	var fields []reflect.StructField
	for i := 0; i < it.NumField(); i++ {
		fields = append(fields, it.Field(i))
	}
	return fields
}
