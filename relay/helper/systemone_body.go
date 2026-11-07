package helper

import (
	"io"

	rootcommon "github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
)

// BuildSystemoneOutboundBody 构造 /v1/systemone 的上游请求体。
//
// 无模型映射（info.IsModelMapped=false，当前渠道 20 的常态）时直接透传客户端
// 原始字节：laya 模型对 JSON 键序敏感，经 map 重序列化（Go 按键名排序）会改变
// 决策答案（2026-11-08 判据 b-fidelity 实锤：直连 0.9063/push_basic_drill，
// 经重序列化 0.9145/live_tutoring）。原始字节从 BodyStorage 取回——校验入口
// UnmarshalBodyReusable 已建立可重放存储，与转发阶段同源。
//
// 配了模型映射时必须改写 body 里的 model 字段，回退到 DeepCopy+Marshal 重序列
// 化路径（与 chat 等形状一致）；该路径的零值保真由
// TestSystemoneRequestForwardRebuildKeepsExplicitZeroValues 守护。
//
// 返回 bodyMode 便于日志区分："raw-passthrough" / "remarshal-mapped"。
func BuildSystemoneOutboundBody(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) (body rootcommon.ReplayableBody, closer io.Closer, bodyMode string, err error) {
	if info.IsModelMapped {
		jsonData, marshalErr := rootcommon.Marshal(request)
		if marshalErr != nil {
			return nil, nil, "", marshalErr
		}
		body, closer, err = relaycommon.NewOutboundJSONBody(jsonData)
		return body, closer, "remarshal-mapped", err
	}

	storage, storageErr := rootcommon.GetBodyStorage(c)
	if storageErr != nil {
		return nil, nil, "", storageErr
	}
	rawBytes, bytesErr := storage.Bytes()
	if bytesErr != nil {
		return nil, nil, "", bytesErr
	}
	body, closer, err = relaycommon.NewOutboundJSONBody(rawBytes)
	return body, closer, "raw-passthrough", err
}
