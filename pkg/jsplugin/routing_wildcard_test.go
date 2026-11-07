package jsplugin

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 通配声明的端点兜底：未被显式声明的模型由通配认领的插件服务；
// 显式声明优先于通配（其绑定必然先命中精确键）。
func TestLookupEndpointCandidatesWildcardClaimFallback(t *testing.T) {
	registry := NewRegistry()
	videoExports := `export const protocols = {
		openai_video: {
			decodeRequest: function(ctx) { return ctx; },
			render: function(ctx, task) { return task; }
		}
	};
	export function listArtifacts(task) { return []; }
	export function buildContentRequest(ctx) { return {url: ctx.baseUrl + "/content", method: "GET"}; }`
	// 通配收窄在协议声明内：openai_video 路径通配认领，openai_responses 路径不受影响
	wildcard := mustCompileRoutingPlugin(t, "wildcard-provider", 1, `["sora-2"]`,
		`protocols: [{name: "openai_video", models: ["*"]}],`, videoExports)
	explicit := mustCompileRoutingPlugin(t, "explicit-provider", 55, `["owned-video"]`,
		`protocols: [{name: "openai_video", models: ["owned-video"]}],`, videoExports)

	require.NoError(t, registry.ReplaceOverrides([]*LoadedPlugin{wildcard, explicit}))
	require.Empty(t, registry.RoutingErrors())
	generation := registry.Generation()
	for _, cl := range wildcard.Meta.Protocols {
		t.Logf("DEBUG claim name=%s models=%v", cl.Name, cl.Models)
	}
	t.Logf("DEBUG wildcard bindings: %d", len(generation.LookupEndpointCandidates("POST", "/v1/videos", "*")))

	// 未被显式声明的模型 → 通配兜底命中
	candidates := generation.LookupEndpointCandidates("POST", "/v1/videos", "Wan2.2-TI2V-5B")
	require.Len(t, candidates, 1)
	assert.Equal(t, "wildcard-provider", candidates[0].Plugin.Meta.Key)

	// 显式声明的模型 → 精确键命中，通配不并列
	owned := generation.LookupEndpointCandidates("POST", "/v1/videos", "owned-video")
	require.Len(t, owned, 1)
	assert.Equal(t, "explicit-provider", owned[0].Plugin.Meta.Key)

	// 通配插件自己的显式条目仍走精确键
	direct := generation.LookupEndpointCandidates("POST", "/v1/videos", "sora-2")
	require.Len(t, direct, 1)
	assert.Equal(t, "wildcard-provider", direct[0].Plugin.Meta.Key)

	// 无通配的协议路径不被劫持：openai_responses 上任意模型零候选
	assert.Empty(t, generation.LookupEndpointCandidates("POST", "/v1/responses", "chat-model"))

	// 两个通配插件争同一路径 → 构建期冲突报错
	second := mustCompileRoutingPlugin(t, "wildcard-provider-2", 56, `["legacy-video"]`,
		`protocols: [{name: "openai_video", models: ["*"]}],`, videoExports)
	// 冲突可能以构建错误返回，也可能记入 RoutingErrors（通道型冲突先行时）
	err := registry.ReplaceOverrides([]*LoadedPlugin{wildcard, second})
	if err != nil {
		assert.Contains(t, err.Error(), "wildcard claim")
	} else {
		errs := registry.RoutingErrors()
		require.NotEmpty(t, errs)
		found := false
		for _, msg := range errs {
			if strings.Contains(msg, "wildcard claim") {
				found = true
			}
		}
		assert.True(t, found, "RoutingErrors 应记录通配冲突: %v", errs)
	}
}
