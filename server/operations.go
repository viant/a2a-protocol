package server

import (
	"context"
	"github.com/viant/jsonrpc"
)

// Operations defines server-side JSON-RPC methods for A2A.
type Operations interface {
	// Message
	MessageSend(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	MessageStream(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)

	// Tasks
	TasksGet(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	TasksCancel(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	TasksResubscribe(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)

	// Push notification config (JSON-RPC)
	TasksPushNotificationConfigSet(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	TasksPushNotificationConfigGet(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	TasksPushNotificationConfigList(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)
	TasksPushNotificationConfigDelete(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)

	// Misc
	AgentGetCard(ctx context.Context, req *jsonrpc.Request, resp *jsonrpc.Response)

	// Notifications
	OnNotification(ctx context.Context, n *jsonrpc.Notification)

	// Implements checks if JSON-RPC method is supported
	Implements(method string) bool
}
