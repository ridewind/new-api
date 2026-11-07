package jsplugin

import (
	"slices"
)

// ModelWildcard is the reserved meta.models entry that declares every model a
// plugin can serve on its channel types. Self-hosted deployments (vLLM-Omni
// and similar OpenAI-compatible task backends) serve arbitrary served model
// names, and requiring one declaration per name would silently bill zero for
// every undeclared video model. A wildcard never enumerates as a concrete
// model: listing and routing indexes skip it.
const ModelWildcard = "*"

// IsModelWildcard reports whether the declared model entry is the wildcard.
// The match is exact: " * " is a (harmless) concrete name, not the wildcard.
func IsModelWildcard(model string) bool {
	return model == ModelWildcard
}

// DeclaresModel reports whether models name model exactly or via the
// wildcard entry.
func DeclaresModel(models []string, model string) bool {
	if slices.Contains(models, model) {
		return true
	}
	return slices.Contains(models, ModelWildcard)
}

// ConcreteModels drops wildcard entries for callers that enumerate declared
// models into listings or routing indexes.
func ConcreteModels(models []string) []string {
	out := make([]string, 0, len(models))
	for _, model := range models {
		if !IsModelWildcard(model) {
			out = append(out, model)
		}
	}
	return out
}
