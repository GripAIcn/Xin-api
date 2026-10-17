package inbound

// chatCodec OpenAI Chat 入站协议：恒等编解码，保证 chat 走同一 handle 分支且行为不变
type chatCodec struct{}

func (c *chatCodec) Name() string { return ProtocolChat }

func (c *chatCodec) NormalizeRequest(body []byte, _ NormalizeOptions) ([]byte, error) {
	return body, nil
}

func (c *chatCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	return openAIResponse, nil
}

func (c *chatCodec) EncodeStreamChunk(openAISSESChunk []byte) ([]byte, error) {
	return openAISSESChunk, nil
}

func (c *chatCodec) LocalError(bizCode int) (int, []byte) {
	return openAILocalError(bizCode)
}

func (c *chatCodec) UpstreamError(upstreamStatus int, upstreamBody []byte) (int, []byte) {
	return upstreamStatus, upstreamBody
}
