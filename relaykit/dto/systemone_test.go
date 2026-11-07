package dto

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemoneRequestUnmarshalKeepsShape(t *testing.T) {
	raw := []byte(`{
		"model": "laya",
		"state": {"checkpoint": "english", "quiz_score": 0},
		"questions": {
			"q1": {"type": "choice", "criteria": {"level": "A2"}},
			"q2": {"type": "score", "range": [0, 100]},
			"q3": {"type": "noul"}
		}
	}`)

	req := &SystemoneRequest{}
	require.NoError(t, json.Unmarshal(raw, req))
	assert.Equal(t, "laya", req.Model)
	// json.Unmarshal 进 any 的数字是 float64
	assert.Equal(t, map[string]any{"checkpoint": "english", "quiz_score": float64(0)}, req.State)
	require.Len(t, req.Questions, 3)
	// 三种题型值结构不同，any/map 承载不重塑
	assert.Equal(t, map[string]any{"type": "choice", "criteria": map[string]any{"level": "A2"}}, req.Questions["q1"])
	assert.Equal(t, map[string]any{"type": "score", "range": []any{float64(0), float64(100)}}, req.Questions["q2"])
	assert.Equal(t, map[string]any{"type": "noul"}, req.Questions["q3"])
}

// 显式零值可选键（lang=false、min_confidence=0、max_len=0）与嵌套零值在
// Unmarshal 再 Marshal 后必须仍在输出体里：omitempty 对非 nil 接口不生效。
func TestSystemoneRequestMarshalKeepsExplicitZeroValues(t *testing.T) {
	raw := []byte(`{
		"model": "laya",
		"state": {"quiz_score": 0},
		"questions": {"q": {"weight": 0}},
		"lang": false,
		"min_confidence": 0,
		"max_len": 0
	}`)

	req := &SystemoneRequest{}
	require.NoError(t, json.Unmarshal(raw, req))

	encoded, err := json.Marshal(req)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(encoded, &out))
	assert.Equal(t, false, out["lang"])
	assert.Equal(t, float64(0), out["min_confidence"])
	assert.Equal(t, float64(0), out["max_len"])
	state, ok := out["state"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), state["quiz_score"])
	questions, ok := out["questions"].(map[string]any)
	require.True(t, ok)
	q, ok := questions["q"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(0), q["weight"])
}

// 模型侧 app.py 逐键读取、未知键忽略，网关侧不报错即中性。
func TestSystemoneRequestUnknownKeysIgnored(t *testing.T) {
	raw := []byte(`{"model":"laya","state":{},"questions":{"q":{}},"future_key":{"a":1}}`)
	req := &SystemoneRequest{}
	require.NoError(t, json.Unmarshal(raw, req))
	assert.Equal(t, "laya", req.Model)
	assert.Len(t, req.Questions, 1)
}

func TestSystemoneRequestMetaAndHelpers(t *testing.T) {
	req := &SystemoneRequest{Model: "laya"}

	// 计费预处理链要求 meta 非 nil（service.CountRequestToken 拒 nil）
	assert.NotNil(t, req.GetTokenCountMeta())
	assert.False(t, req.IsStream(httptest.NewRequest("POST", "/v1/systemone", nil)))

	req.SetModelName("")
	assert.Equal(t, "laya", req.Model, "空串不覆盖")
	req.SetModelName("mapped-laya")
	assert.Equal(t, "mapped-laya", req.Model)
}
