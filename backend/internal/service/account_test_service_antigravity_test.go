//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAntigravityConnectionTest_VisibleOutput(t *testing.T) {
	const textResponse = `data: {"response":{"candidates":[{"content":{"parts":[{"text":"OK."}]},"finishReason":"STOP"}]}}` + "\n\n"
	const imageResponse = `data: {"response":{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"QUJD"}}]}}]}}` + "\n\n"
	for _, tc := range []struct {
		name     string
		model    string
		prompt   string
		response string
		wantErr  bool
		image    bool
	}{
		{name: "Gemini confirmation", model: "gemini-3-flash", response: textResponse},
		{name: "Claude confirmation", model: "claude-sonnet-4-6", response: textResponse},
		{name: "Gemini custom prompt", model: "gemini-3-flash", prompt: "  Say hello.  ", response: textResponse},
		{name: "Claude custom prompt", model: "claude-sonnet-4-6", prompt: `Say "hello".`, response: textResponse},
		{name: "output budget exhausted", model: "gemini-3-flash", response: `data: {"response":{"candidates":[{"content":{"parts":[{"text":""}]},"finishReason":"MAX_TOKENS"}]}}` + "\n\n", wantErr: true},
		{name: "whitespace only", model: "gemini-3-flash", response: `data: {"response":{"candidates":[{"content":{"parts":[{"text":" "}]},"finishReason":"STOP"}]}}` + "\n\n", wantErr: true},
		{name: "image only", model: "gemini-3.1-flash-image", response: imageResponse, image: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := &queuedHTTPUpstreamStub{responses: []*http.Response{{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(tc.response)),
			}}}
			svc := &AntigravityGatewayService{
				tokenProvider: &AntigravityTokenProvider{},
				httpUpstream:  upstream,
			}
			account := &Account{
				ID: 1, Platform: PlatformAntigravity, Type: AccountTypeOAuth,
				Concurrency: 1,
				Credentials: map[string]any{
					"access_token": "test-token", "project_id": "test-project",
					"model_mapping": map[string]any{tc.model: tc.model},
				},
			}
			result, err := svc.TestConnectionWithPrompt(context.Background(), account, tc.model, tc.prompt)
			if tc.wantErr {
				require.ErrorContains(t, err, "未返回文本或图片")
				require.Nil(t, result, "empty responses must not be reported as successful tests")
			} else {
				require.NoError(t, err)
				if tc.image {
					require.Len(t, result.Images, 1)
					require.Empty(t, result.Text)
				} else {
					require.Equal(t, "OK.", result.Text)
				}
			}
			require.Len(t, upstream.requestBodies, 1)
			var request struct {
				Request struct {
					GenerationConfig struct {
						MaxOutputTokens int `json:"maxOutputTokens"`
					} `json:"generationConfig"`
					Contents []struct {
						Parts []struct {
							Text string `json:"text"`
						} `json:"parts"`
					} `json:"contents"`
				} `json:"request"`
			}
			require.NoError(t, json.Unmarshal(upstream.requestBodies[0], &request))
			if !tc.image {
				require.GreaterOrEqual(t, request.Request.GenerationConfig.MaxOutputTokens, 1024)
				wantPrompt := strings.TrimSpace(tc.prompt)
				if wantPrompt == "" {
					wantPrompt = "Reply with exactly OK."
				}
				require.Equal(t, wantPrompt, request.Request.Contents[0].Parts[0].Text)
			}
		})
	}
}

func TestAntigravityConnectionTestModel(t *testing.T) {
	t.Parallel()

	require.Equal(t, "claude-sonnet-4-6", antigravityConnectionTestModel(""))
	require.Equal(t, "gemini-3.1-pro-preview", antigravityConnectionTestModel("gemini-3.1-pro-preview"))
}

func TestAntigravityBuildGeminiTestRequest_IncludesValidatedToolConfig(t *testing.T) {
	t.Parallel()

	svc := &AntigravityGatewayService{}
	raw, err := svc.buildGeminiTestRequestWithPrompt("proj-123", "gemini-3.8-flash-high", "")
	require.NoError(t, err)

	var v1Req map[string]any
	require.NoError(t, json.Unmarshal(raw, &v1Req))
	require.Equal(t, "proj-123", v1Req["project"])
	require.Equal(t, "gemini-3.8-flash-high", v1Req["model"])

	inner, ok := v1Req["request"].(map[string]any)
	require.True(t, ok)
	toolCfg, ok := inner["toolConfig"].(map[string]any)
	require.True(t, ok, "expected toolConfig to be present in Gemini test request")
	fcCfg, ok := toolCfg["functionCallingConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "VALIDATED", fcCfg["mode"])
}

func TestAntigravityBuildGeminiTestRequestWithPrompt_ImageModel(t *testing.T) {
	t.Parallel()

	svc := &AntigravityGatewayService{}
	raw, err := svc.buildGeminiTestRequestWithPrompt("proj-123", "gemini-3.1-flash-image", "a cute cat")
	require.NoError(t, err)

	var v1Req map[string]any
	require.NoError(t, json.Unmarshal(raw, &v1Req))
	inner, ok := v1Req["request"].(map[string]any)
	require.True(t, ok)

	genCfg, ok := inner["generationConfig"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, []any{"TEXT", "IMAGE"}, genCfg["responseModalities"])

	contents, ok := inner["contents"].([]any)
	require.True(t, ok)
	require.Len(t, contents, 1)
	firstContent := contents[0].(map[string]any)
	parts := firstContent["parts"].([]any)
	require.Equal(t, "a cute cat", parts[0].(map[string]any)["text"])
}

func TestExtractContentFromSSEResponse_ExtractsTextAndInlineImages(t *testing.T) {
	t.Parallel()

	sse := "data: {\"response\":{\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Here is your image\"},{\"inlineData\":{\"mimeType\":\"image/png\",\"data\":\"iVBORw0KGgo=\"}}]}}]}}\n\n"
	text, images := extractContentFromSSEResponse([]byte(sse))
	require.Equal(t, "Here is your image", text)
	require.Len(t, images, 1)
	require.Equal(t, "image/png", images[0].MimeType)
	require.Equal(t, "data:image/png;base64,iVBORw0KGgo=", images[0].ImageURL)
}

func TestAntigravityShouldUseGeminiTestBuilder_UsesMappedModel(t *testing.T) {
	t.Parallel()

	require.True(t, shouldUseAntigravityGeminiTestRequest("custom-alias", "gemini-2.5-flash"))
	require.True(t, shouldUseAntigravityGeminiTestRequest("gemini-3.1-pro", "gemini-pro-agent"))
	require.False(t, shouldUseAntigravityGeminiTestRequest("gemini-alias", "claude-sonnet-4-6"))
	require.False(t, shouldUseAntigravityGeminiTestRequest("claude-sonnet-4-6", "claude-sonnet-4-6"))
}
