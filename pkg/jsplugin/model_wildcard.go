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

// HasModelWildcard reports whether the plugin claims models via a wildcard —
// at the top level or inside any protocol claim.
func (m Meta) HasModelWildcard() bool {
	if slices.Contains(m.Models, ModelWildcard) {
		return true
	}
	for _, claim := range m.Protocols {
		if slices.Contains(claim.Models, ModelWildcard) {
			return true
		}
	}
	return false
}

// DeclaresModelName is the model-ownership check for a whole plugin: the top
// level list, any protocol claim's list, or any wildcard in either.
func (m Meta) DeclaresModelName(name string) bool {
	if slices.Contains(m.Models, name) || slices.Contains(m.Models, ModelWildcard) {
		return true
	}
	for _, claim := range m.Protocols {
		if slices.Contains(claim.Models, name) || slices.Contains(claim.Models, ModelWildcard) {
			return true
		}
	}
	return false
}

// EffectiveModelScope is the model list ownership checks run against: the
// top level list, every protocol claim's list, and the wildcard when the
// plugin claims one anywhere.
func (m Meta) EffectiveModelScope() []string {
	out := slices.Clone(m.Models)
	for _, claim := range m.Protocols {
		out = append(out, claim.Models...)
	}
	if slices.Contains(out, ModelWildcard) {
		return out
	}
	if m.HasModelWildcard() {
		return append(out, ModelWildcard)
	}
	return out
}
