package relay

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// VideoSyncHelper 承载 POST /v1/videos/sync（同步视频生成，multipart 进、
// MP4 二进制出）。请求表单按原样重建转发（含全部文件字段）；
// 计费在 VideoSyncHandler 里按响应头 x-inference-time-s 折算秒数，
// 走 PostAudioConsumeQuota：配了 ModelPrice 按次，配了音频倍率按秒。
func VideoSyncHelper(c *gin.Context, info *relaycommon.RelayInfo) (newAPIError *types.NewAPIError) {
	info.InitChannelMeta(c)

	videoReq, ok := info.Request.(*dto.VideoSyncRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.VideoSyncRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(videoReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to VideoSyncRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	// LiteLLM 直通层按 query 参数选部署（multipart 体不解析），
	// 上游 URL 必须带模型名；客户端已带 ?model= 时不重复加
	if !strings.Contains(info.RequestURLPath, "model=") {
		sep := "?"
		if strings.Contains(info.RequestURLPath, "?") {
			sep = "&"
		}
		info.RequestURLPath = info.RequestURLPath + sep + "model=" + url.QueryEscape(request.Model)
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	// 重建 multipart：全部字段 + 全部文件（校验阶段已把表单挂回 c.Request）
	mf := c.Request.MultipartForm
	if mf == nil {
		return types.NewErrorWithStatusCode(fmt.Errorf("multipart form is not parsed"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	for key, values := range mf.Value {
		for _, value := range values {
			if key == "model" {
				// 渠道映射后的上游模型名
				value = request.Model
			}
			if err := writer.WriteField(key, value); err != nil {
				return types.NewError(fmt.Errorf("failed to write form field %s: %w", key, err), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
		}
	}
	for field, fileHeaders := range mf.File {
		for _, fileHeader := range fileHeaders {
			part, err := writer.CreateFormFile(field, fileHeader.Filename)
			if err != nil {
				return types.NewError(fmt.Errorf("failed to create form file %s: %w", field, err), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			file, err := fileHeader.Open()
			if err != nil {
				return types.NewError(fmt.Errorf("failed to open form file %s: %w", field, err), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			if _, err := io.Copy(part, file); err != nil {
				file.Close()
				return types.NewError(fmt.Errorf("failed to copy form file %s: %w", field, err), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			file.Close()
		}
	}
	if err := writer.Close(); err != nil {
		return types.NewError(fmt.Errorf("failed to close multipart writer: %w", err), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
	}
	// DoFormRequest 按当前 c.Request.Header 的 Content-Type（含 boundary）发请求
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	logger.LogDebug(c, "video sync request body size: %d bytes, model: %s", requestBody.Len(), request.Model)

	resp, err := adaptor.DoRequest(c, info, &requestBody)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}

	statusCodeMappingStr := c.GetString("status_code_mapping")
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			newAPIError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
			service.ResetStatusCode(newAPIError, statusCodeMappingStr)
			return newAPIError
		}
	}

	usage, newAPIError := adaptor.DoResponse(c, httpResp, info)
	if newAPIError != nil {
		service.ResetStatusCode(newAPIError, statusCodeMappingStr)
		return newAPIError
	}
	service.PostAudioConsumeQuota(c, info, usage.(*dto.Usage), "")
	return nil
}
