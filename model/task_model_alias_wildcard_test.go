package model

import (
	"testing"

	"github.com/QuantumNous/new-api/pkg/jsplugin"
)

func TestResolveTaskPluginModelWildcard(t *testing.T) {
	models := []string{"sora-2", jsplugin.ModelWildcard}
	served, declared, ok := ResolveTaskPluginModel(nil, "Any-Served-Name", models)
	if !ok || served != "Any-Served-Name" || declared != "Any-Served-Name" {
		t.Fatalf("wildcard resolve=(%q,%q,%v)", served, declared, ok)
	}
	served, declared, ok = ResolveTaskPluginModel(nil, "sora-2", models)
	if !ok || served != "sora-2" || declared != "sora-2" {
		t.Fatalf("exact resolve=(%q,%q,%v)", served, declared, ok)
	}
	if _, _, ok := ResolveTaskPluginModel(nil, "Any-Served-Name", []string{"sora-2"}); ok {
		t.Fatal("without wildcard an undeclared model must not resolve")
	}
}
