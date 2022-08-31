// Code generated by go generate. DO NOT EDIT.

package opt

import (
	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
)

// ExtractAttributesToHighlight returns the first found AttributesToHighlightOption from the
// given variadic arguments or nil otherwise.
func ExtractAttributesToHighlight(opts ...interface{}) *opt.AttributesToHighlightOption {
	for _, o := range opts {
		if v, ok := o.(*opt.AttributesToHighlightOption); ok {
			return v
		}
	}
	return nil
}