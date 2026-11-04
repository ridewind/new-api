package dto

import (
	"net/http"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// OCRRequest 是 /v1/ocr 的请求体。aihub 自研推理镜像（cpu-ocr）的约定：
// JSON 携带 image / image_base64 / file 三者之一（base64 或 data URI），
// model 字段由网关用于渠道路由，原样转发给上游。
type OCRRequest struct {
	Model       string `json:"model"`
	Image       any    `json:"image,omitempty"`
	ImageBase64 string `json:"image_base64,omitempty"`
	File        string `json:"file,omitempty"`
}

func (r *OCRRequest) IsStream(c *http.Request) bool {
	return false
}

// GetTokenCountMeta 返回零值 meta：OCR 按次计费（ModelPrice），没有 token
// 估算诉求，但计费预处理链要求 meta 非空（service.CountRequestToken 拒 nil）。
func (r *OCRRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return &types.TokenCountMeta{}
}

func (r *OCRRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

func (r *OCRRequest) HasImage() bool {
	return r.Image != nil || r.ImageBase64 != "" || r.File != ""
}
