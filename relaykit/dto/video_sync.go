package dto

import (
	"net/http"

	"github.com/QuantumNous/new-api/relaykit/types"
)

// VideoSyncRequest 是 POST /v1/videos/sync（multipart）的请求抽象。
// 表单字段（prompt/width/height/aspect_ratio/fps/num_inference_steps/
// flow_shift/audio_flow_shift/seed/extra_params 等）不在此结构体逐一定义，
// 转发时直接按 c.Request.MultipartForm 原样重建 multipart；这里只承接
// 计费与路由需要的字段。
type VideoSyncRequest struct {
	Model string `json:"model"`
	// DurationSeconds 是计费兜底用的请求时长（秒）：优先取 extra_params
	// JSON 里的 duration，其次顶层 duration 表单字段。响应头
	// x-inference-time-s（真实推理秒数）存在时以其为准。
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
}

func (r *VideoSyncRequest) IsStream(c *http.Request) bool {
	return false
}

// GetTokenCountMeta 返回 nil：同步视频按秒（音频 token 机制）或按次计费。
func (r *VideoSyncRequest) GetTokenCountMeta() *types.TokenCountMeta {
	return nil
}

func (r *VideoSyncRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}
