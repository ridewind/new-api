package jsplugin

import (
	"slices"
	"testing"
)

func TestDeclaresModel(t *testing.T) {
	declared := []string{"sora-2", "Wan2.2-TI2V-5B", ModelWildcard}
	cases := []struct {
		name  string
		model string
		want  bool
	}{
		{"exact declared", "sora-2", true},
		{"other declared", "Wan2.2-TI2V-5B", true},
		{"undeclared hits wildcard", "MiniMax-H3", true},
	}
	list := declared
	for _, tc := range cases {
		if got := DeclaresModel(list, tc.model); got != tc.want {
			t.Errorf("%s: DeclaresModel=%v want %v", tc.name, got, tc.want)
		}
	}
	if DeclaresModel([]string{"sora-2"}, "MiniMax-H3") {
		t.Error("no wildcard entry must not declare other models")
	}
	if DeclaresModel(nil, "sora-2") {
		t.Error("nil list declares nothing")
	}
}

func TestConcreteModels(t *testing.T) {
	got := ConcreteModels([]string{"sora-2", " ", ModelWildcard, "Wan2.2-TI2V-5B"})
	if !slices.Equal(got, []string{"sora-2", " ", "Wan2.2-TI2V-5B"}) {
		t.Errorf("ConcreteModels=%v", got)
	}
	if got := ConcreteModels(nil); len(got) != 0 {
		t.Errorf("ConcreteModels(nil)=%v", got)
	}
	// 一个仅含空白的条目不是通配符：通配必须是恰好 "*"
	if IsModelWildcard(" * ") {
		t.Error("padded wildcard must not match; only exact \"*\" is the wildcard")
	}
}
