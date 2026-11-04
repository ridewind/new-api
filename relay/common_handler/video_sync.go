package common_handler

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// VideoSyncHandler 把上游同步视频响应（MP4 二进制）透传给客户端。
// 计费：优先读响应头 x-inference-time-s（真实推理秒数），其次请求里
// 估算的时长，折算为音频 token（每分钟 1000，与 TTS 同机制）后由
// PostAudioConsumeQuota 结算——模型配 ModelPrice 则按次，配音频倍率则按秒。
func VideoSyncHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response, request *dto.VideoSyncRequest) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)
	usage := &dto.Usage{}

	// 透传安全的上游响应头（含 x-inference-time-s 等计费/观测头）
	for k, v := range resp.Header {
		if !service.ShouldCopyUpstreamHeader(c, k, v) {
			continue
		}
		c.Writer.Header().Set(k, v[0])
	}
	c.Writer.WriteHeader(resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(c, fmt.Sprintf("failed to read video sync response body: %v", err))
		c.Writer.WriteHeaderNow()
		return usage, nil
	}
	c.Writer.WriteHeaderNow()
	if _, err := c.Writer.Write(bodyBytes); err != nil {
		logger.LogError(c, fmt.Sprintf("failed to write video sync response: %v", err))
	}

	seconds := 0.0
	for _, header := range []string{"X-Inference-Time-S", "x-inference-time-s", "X-Video-Duration-S", "X-Audio-Duration"} {
		if raw := resp.Header.Get(header); raw != "" {
			if v, err := strconv.ParseFloat(raw, 64); err == nil && v > 0 {
				seconds = v
				break
			}
		}
	}
	if seconds == 0 && request != nil {
		seconds = request.DurationSeconds
	}

	if seconds > 0 {
		// 与 TTS 同口径：ceil(秒)/60*1000 音频 token
		tokens := common.QuotaRound(math.Ceil(seconds) / 60.0 * 1000)
		usage.CompletionTokens = tokens
		usage.CompletionTokenDetails.AudioTokens = tokens
		usage.TotalTokens = tokens
		common.SetContextKey(c, "video_sync_seconds", seconds)
	}
	logger.LogInfo(c, fmt.Sprintf("video sync completed: %d bytes, billed seconds: %v, audio tokens: %d", len(bodyBytes), seconds, usage.CompletionTokenDetails.AudioTokens))
	return usage, nil
}
