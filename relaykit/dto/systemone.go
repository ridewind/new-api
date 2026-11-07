package dto

import (
	"net/http"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// SystemoneRequest 是 /v1/systemone 的请求体（System One 原生决策形状）。
// state 是任意 JSON、questions 值结构随题型变化（choice 的 criteria 是
// dict、score 是 list、noul 无），因此用 any / map 承载原样透传，不做
// 强类型重塑；可选参数为 cpu-laya app.py 的六个可选键，原样转发。
type SystemoneRequest struct {
	Model         string         `json:"model"`
	State         any            `json:"state,omitempty"`
	Questions     map[string]any `json:"questions,omitempty"`
	Task          any            `json:"task,omitempty"`
	Lang          any            `json:"lang,omitempty"`
	LangGuess     any            `json:"lang_guess,omitempty"`
	MaxLen        any            `json:"max_len,omitempty"`
	HeadMaxLen    any            `json:"head_max_len,omitempty"`
	MinConfidence any            `json:"min_confidence,omitempty"`
}

func (r *SystemoneRequest) IsStream(c *http.Request) bool {
	return false
}

// GetTokenCountMeta 返回零值 meta：systemone 按次计费（ModelPrice），
// 没有 token 估算诉求，但计费预处理链要求 meta 非 nil
// （service.CountRequestToken 拒 nil，同 OCR 先例）。
func (r *SystemoneRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{}
}

func (r *SystemoneRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
