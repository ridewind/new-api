package dto

import "testing"

func TestRerankResponseMetaFillUsage(t *testing.T) {
	cases := []struct {
		name string
		json string
		want PromptCompletion
	}{
		{
			name: "litellm hosted_vllm 形状：billed_units.total_tokens 与 tokens.input_tokens",
			json: `{"results":[{"index":0,"relevance_score":0.9}],"usage":{},"meta":{"billed_units":{"total_tokens":120},"tokens":{"input_tokens":120}}}`,
			want: PromptCompletion{120, 0},
		},
		{
			name: "真 Cohere 形状：billed_units.input+output",
			json: `{"results":[],"usage":{},"meta":{"billed_units":{"input_tokens":100,"output_tokens":20}}}`,
			want: PromptCompletion{100, 20},
		},
		{
			name: "只有 tokens.input+output",
			json: `{"results":[],"usage":{},"meta":{"tokens":{"input_tokens":7,"output_tokens":3}}}`,
			want: PromptCompletion{7, 3},
		},
		{
			name: "顶层 usage 已有值时不覆盖",
			json: `{"results":[],"usage":{"prompt_tokens":5,"total_tokens":9},"meta":{"billed_units":{"total_tokens":100}}}`,
			want: PromptCompletion{5, 0},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var resp RerankResponse
			if err := unmarshalForTest(tc.json, &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			resp.Meta.FillUsage(&resp.Usage)
			if resp.Usage.PromptTokens != tc.want.Prompt || resp.Usage.CompletionTokens != tc.want.Completion {
				t.Fatalf("got prompt=%d completion=%d total=%d, want prompt=%d completion=%d",
					resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
					tc.want.Prompt, tc.want.Completion)
			}
		})
	}
}

type PromptCompletion struct {
	Prompt     int
	Completion int
}
