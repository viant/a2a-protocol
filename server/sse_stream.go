package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/viant/jsonrpc"
	base "github.com/viant/jsonrpc/transport/server/base"
)

// sendSSEResponse encodes result as a JSON-RPC response and pushes it
// over the SSE stream associated with the current session.
func sendSSEResponse(ctx context.Context, result interface{}) error {
	sessVal := ctx.Value(jsonrpc.SessionKey)
	s, ok := sessVal.(*base.Session)
	if !ok || s == nil {
		return fmt.Errorf("sse session not found in context")
	}
	// Try to get the originating request id; fall back to last id.
	var id jsonrpc.RequestId
	if v := ctx.Value(jsonrpc.RequestIdKey); v != nil {
		if rid, ok := v.(jsonrpc.RequestId); ok {
			id = rid
		}
	}
	if id == nil {
		id = s.LastRequestID()
	}
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	resp := &jsonrpc.Response{Id: id, Jsonrpc: jsonrpc.Version, Result: data}
	s.SendResponse(ctx, resp)
	return nil
}
