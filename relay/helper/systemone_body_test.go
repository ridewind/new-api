package helper

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	rootcommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildForwardContext 复刻真实链路：InitChannelMeta（分配 ChannelMeta）→
// GetAndValidateSystemoneRequest（内部 UnmarshalBodyReusable 建立 BodyStorage）
// → 转发阶段取回原始字节。
func buildForwardContext(t *testing.T, body string, modelMapping string) (*gin.Context, *relaycommon.RelayInfo, dto.Request) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if modelMapping != "" {
		c.Set("model_mapping", modelMapping)
	}
	// InitChannelMeta 依赖渠道上下文键，缺省值为 0 即可分配 ChannelMeta
	c.Set(string(constant.ContextKeyChannelType), 0)
	c.Set(string(constant.ContextKeyOriginalModel), "laya")

	req, err := GetAndValidateSystemoneRequest(c)
	require.NoError(t, err)

	info := relaycommon.GenRelayInfoSystemone(c, req)
	info.InitChannelMeta(c)
	return c, info, req
}

// TestSystemoneRawPassthroughKeepsOriginalBytes 是键序保真的仓内权威验证：
// 无模型映射时转发字节必须与客户端原始字节逐字节一致（含键序）。laya 模型对
// 键序敏感，Go map 重序列化按字母排序会改变决策答案（2026-11-08 判据
// b-fidelity 实锤），本用例防止该路径被改回 DeepCopy+Marshal。
func TestSystemoneRawPassthroughKeepsOriginalBytes(t *testing.T) {
	// 故意用非字母序键序：questions 在 state 前、choice 在 type 前
	body := `{"questions":{"q_next_action":{"type":"choice","criteria":{"push_basic_drill":"巩固基础","live_tutoring":"答疑"}},"q_push_practice":{"type":"noul"}},"state":{"checkpoint":"english"},"model":"laya"}`

	c, info, req := buildForwardContext(t, body, "")

	out, closer, bodyMode, err := BuildSystemoneOutboundBody(c, info, req)
	require.NoError(t, err)
	defer closer.Close()
	assert.Equal(t, "raw-passthrough", bodyMode)

	raw, err := io.ReadAll(out)
	require.NoError(t, err)
	assert.Equal(t, body, string(raw), "转发字节必须与原始请求字节逐字节一致")
}

// TestSystemoneRemarshalWhenModelMapped 验证配了模型映射的回退路径：此时必须
// 改写 body 里的 model 字段，走重序列化（与 chat 等形状行为一致）。
func TestSystemoneRemarshalWhenModelMapped(t *testing.T) {
	body := `{"model":"laya","state":{"checkpoint":"english"},"questions":{"q":{"type":"noul"}}}`

	c, info, req := buildForwardContext(t, body, `{"laya":"laya-served"}`)
	// 复刻 SystemoneHelper 的真实顺序：先映射再构造 body
	err := ModelMappedHelper(c, info, req)
	require.NoError(t, err)

	out, closer, bodyMode, err := BuildSystemoneOutboundBody(c, info, req)
	require.NoError(t, err)
	defer closer.Close()
	assert.Equal(t, "remarshal-mapped", bodyMode)
	assert.True(t, info.IsModelMapped)

	raw, err := io.ReadAll(out)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, rootcommon.Unmarshal(raw, &m))
	assert.Equal(t, "laya-served", m["model"], "映射路径必须把 model 改成渠道 served 名")
	assert.Equal(t, "laya-served", info.UpstreamModelName)
}
