package dto

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/types"
)

type RerankRequest struct {
	Documents       []any  `json:"documents"`
	Query           string `json:"query"`
	Model           string `json:"model"`
	TopN            *int   `json:"top_n,omitempty"`
	ReturnDocuments *bool  `json:"return_documents,omitempty"`
	MaxChunkPerDoc  *int   `json:"max_chunk_per_doc,omitempty"`
	OverLapTokens   *int   `json:"overlap_tokens,omitempty"`
}

func (r *RerankRequest) IsStream(c *http.Request) bool {
	return false
}

func (r *RerankRequest) GetTokenCountMeta() *types.TokenCountMeta {
	var texts = make([]string, 0)

	for _, document := range r.Documents {
		texts = append(texts, fmt.Sprintf("%v", document))
	}

	if r.Query != "" {
		texts = append(texts, r.Query)
	}

	return &types.TokenCountMeta{
		CombineText: strings.Join(texts, "\n"),
	}
}

func (r *RerankRequest) SetModelName(modelName string) {
	if modelName != "" {
		r.Model = modelName
	}
}

func (r *RerankRequest) GetReturnDocuments() bool {
	if r.ReturnDocuments == nil {
		return false
	}
	return *r.ReturnDocuments
}

type RerankResponseResult struct {
	Document       any     `json:"document,omitempty"`
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevance_score"`
}

type RerankDocument struct {
	Text any `json:"text"`
}

type RerankResponse struct {
	Results []RerankResponseResult `json:"results"`
	Usage   Usage                  `json:"usage"`
	// Cohere 兼容上游（如 LiteLLM hosted_vllm 适配层）把用量放在 meta 里而不是顶层 usage，
	// 解析后用于计费回填，同时原样透传给客户端。
	Meta *RerankResponseMeta `json:"meta,omitempty"`
}

type RerankBilledUnits struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
	SearchUnits  int `json:"search_units,omitempty"`
	TotalTokens  int `json:"total_tokens,omitempty"`
}

type RerankTokensMeta struct {
	InputTokens  int `json:"input_tokens,omitempty"`
	OutputTokens int `json:"output_tokens,omitempty"`
}

type RerankResponseMeta struct {
	BilledUnits *RerankBilledUnits `json:"billed_units,omitempty"`
	Tokens      *RerankTokensMeta  `json:"tokens,omitempty"`
}

// FillUsage 在顶层 usage 缺失（TotalTokens==0）时从 Cohere 形状的 meta 回填用量。
// 查找次序：billed_units.input+output → billed_units.total → tokens.input+output。
// LiteLLM v1.104.1 的 hosted_vllm 适配层只填 billed_units.total_tokens 与
// tokens.input_tokens（都取自上游 total_tokens），两条路径都要覆盖。
func (m *RerankResponseMeta) FillUsage(u *Usage) {
	if m == nil || u == nil || u.TotalTokens != 0 {
		return
	}
	in, out := 0, 0
	if m.BilledUnits != nil {
		in, out = m.BilledUnits.InputTokens, m.BilledUnits.OutputTokens
		if in+out == 0 && m.BilledUnits.TotalTokens > 0 {
			in = m.BilledUnits.TotalTokens
		}
	}
	if in+out == 0 && m.Tokens != nil {
		in, out = m.Tokens.InputTokens, m.Tokens.OutputTokens
	}
	if in+out > 0 {
		u.PromptTokens = in
		u.CompletionTokens = out
		u.TotalTokens = in + out
	}
}
