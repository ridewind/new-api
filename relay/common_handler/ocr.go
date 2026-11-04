package common_handler

import (
	"io"
	"net/http"

	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// OCRHandler 把上游 /v1/ocr 响应透传给客户端，并返回零 usage
// （OCR 按次计费，走 ModelPrice 分支，与 token 无关）。
func OCRHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	logger.LogDebug(c, "ocr response body size: %d bytes", len(responseBody))

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.WriteHeader(http.StatusOK)
	if _, err := c.Writer.Write(responseBody); err != nil {
		logger.LogError(c, "failed to write ocr response: "+err.Error())
	}

	usage := &dto.Usage{}
	return usage, nil
}
