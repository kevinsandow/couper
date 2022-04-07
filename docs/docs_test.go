package docs_test

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/avenga/couper/internal/test"
)

func TestDocs_Links(t *testing.T) {
	helper := test.New(t)

	files, err := os.ReadDir("./")
	helper.Must(err)

	type item struct {
		reference string
		value     string
	}

	type entry struct {
		anchors, htmlAnchors, links []item
	}

	entries := map[string]*entry{}

	var existsFn func([]item, item) bool
	existsFn = func(anchors []item, link item) bool {
		if link.reference != "" {
			e, ok := entries[link.reference]
			if !ok {
				helper.Must(fmt.Errorf("missing file: %v", link))
			}
			return existsFn(e.anchors, item{value: link.value})
		}
		var exist bool
		for _, anchor := range anchors {
			exist = anchor.value == link.value
			if exist {
				break
			}
		}
		return exist
	}

	for _, file := range files {
		if file.IsDir() || !isMarkdownFilename(file.Name()) {
			continue
		}
		raw, readErr := os.ReadFile(file.Name())
		helper.Must(readErr)

		regexLinks := regexp.MustCompile(`]\((\./)?(\w+\.md)?#([^)]+)\)`)
		allLinks := regexLinks.FindAllStringSubmatch(string(raw), -1)
		var links []item
		for _, match := range allLinks {
			links = append(links, item{reference: match[2], value: match[3]})
		}

		regexAnchors := regexp.MustCompile(`(?m)^#+ (.+)$`)
		allAnchors := regexAnchors.FindAllStringSubmatch(string(raw), -1)
		var anchors []item
		for _, match := range allAnchors {
			anchors = append(anchors, item{value: prepareAnchor(match[1])})
		}

		regexHTMLAnchors := regexp.MustCompile(`(?m)^<a id="([^"]+)"></a>$`)
		allHTMLAnchors := regexHTMLAnchors.FindAllStringSubmatch(string(raw), -1)
		var htmlAnchors []item
		for _, match := range allHTMLAnchors {
			htmlAnchors = append(htmlAnchors, item{value: match[1]})
		}

		entries[file.Name()] = &entry{
			anchors:     anchors,
			htmlAnchors: htmlAnchors,
			links:       links,
		}
	}

	for filename, file := range entries {
		// Search for ghost-anchors
		for _, link := range file.links {
			if !existsFn(file.anchors, link) && !existsFn(file.htmlAnchors, link) {
				val := link.value
				if link.reference != "" {
					val = link.reference + ":" + val
				}
				t.Errorf("%s: anchor for link %q not found", filename, val)
			}
		}

		// Search for ghost-links - ignore HTML anchors added specifically for VS Code.
		for _, anchor := range file.anchors {
			if anchor.value != "table-of-contents" && !existsFn(file.links, anchor) {
				t.Errorf("%s: link for '%v' not found", filename, anchor)
			}
		}
	}
}

func prepareAnchor(anchor string) string {
	anchor = strings.TrimSpace(anchor)
	anchor = strings.ToLower(anchor)
	anchor = strings.ReplaceAll(anchor, "`", "")
	anchor = strings.ReplaceAll(anchor, ".", "")
	anchor = strings.ReplaceAll(anchor, ":", "")
	anchor = strings.ReplaceAll(anchor, "(", "")
	anchor = strings.ReplaceAll(anchor, ")", "")
	anchor = strings.ReplaceAll(anchor, " ", "-")

	return anchor
}

func isMarkdownFilename(name string) bool {
	return strings.HasSuffix(name, ".md")
}
