package helper

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAndValidateSystemoneRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, body string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}

	t.Run("合法体通过", func(t *testing.T) {
		c := newContext(t, `{"model":"laya","state":{"checkpoint":"english"},"questions":{"q1":{"type":"noul"}}}`)
		req, err := GetAndValidateSystemoneRequest(c)
		require.NoError(t, err)
		assert.Equal(t, "laya", req.Model)
		assert.NotNil(t, req.State)
		assert.Len(t, req.Questions, 1)
	})

	t.Run("缺 model", func(t *testing.T) {
		c := newContext(t, `{"state":{},"questions":{"q":{}}}`)
		_, err := GetAndValidateSystemoneRequest(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "model is required")
	})

	t.Run("缺 state", func(t *testing.T) {
		c := newContext(t, `{"model":"laya","questions":{"q":{}}}`)
		_, err := GetAndValidateSystemoneRequest(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "state is required")
	})

	t.Run("questions 为空对象", func(t *testing.T) {
		c := newContext(t, `{"model":"laya","state":{},"questions":{}}`)
		_, err := GetAndValidateSystemoneRequest(c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "questions is required")
	})

	t.Run("带全部可选键完整通过", func(t *testing.T) {
		c := newContext(t, `{"model":"laya","state":{},"questions":{"q":{}},"task":"assess","lang":"en","lang_guess":true,"max_len":64,"head_max_len":8,"min_confidence":0.5}`)
		req, err := GetAndValidateSystemoneRequest(c)
		require.NoError(t, err)
		assert.Equal(t, "assess", req.Task)
		assert.Equal(t, "en", req.Lang)
		assert.Equal(t, true, req.LangGuess)
		assert.Equal(t, float64(64), req.MaxLen)
		assert.Equal(t, float64(8), req.HeadMaxLen)
		assert.Equal(t, 0.5, req.MinConfidence)
	})
}

// TestSystemoneRequestForwardRebuildKeepsExplicitZeroValues 是 systemone
// 请求保真结论的仓内权威验证：显式零值可选键（lang=false、min_confidence=0、
// max_len=0）与嵌套零值（state.quiz_score=0、questions.q.weight=0）经真实
// common.UnmarshalBodyReusable → common.DeepCopy → common.Marshal 后原样保留。
// 若未来把可选键从 any 改成 bool/float64 等类型化字段，零值恒等无法与被跳过
// 区分，本用例会失败（copier IgnoreEmpty 会跳过类型化零值字段）。
func TestSystemoneRequestForwardRebuildKeepsExplicitZeroValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"model":"laya","state":{"checkpoint":"english","quiz_score":0},"questions":{"q":{"weight":0}},"lang":false,"min_confidence":0,"max_len":0}`
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/systemone", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")

	req, err := GetAndValidateSystemoneRequest(c)
	require.NoError(t, err)

	copied, err := common.DeepCopy(req)
	require.NoError(t, err)

	encoded, err := common.Marshal(copied)
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
	assert.Equal(t, "laya", out["model"])
}
