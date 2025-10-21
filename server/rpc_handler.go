package server

import (
	"context"
	"github.com/viant/jsonrpc"
	"github.com/viant/jsonrpc/transport"
)

// a2aHandler adapts Operations to a jsonrpc transport.Handler.
type a2aHandler struct{ ops Operations }

func (h *a2aHandler) Serve(ctx context.Context, request *jsonrpc.Request, response *jsonrpc.Response) {
	response.Id = request.Id
	response.Jsonrpc = jsonrpc.Version
	switch request.Method {
	case "message/send":
		h.ops.MessageSend(ctx, request, response)
	case "message/stream":
		h.ops.MessageStream(ctx, request, response)
	case "tasks/get":
		h.ops.TasksGet(ctx, request, response)
	case "tasks/cancel":
		h.ops.TasksCancel(ctx, request, response)
	case "tasks/resubscribe":
		h.ops.TasksResubscribe(ctx, request, response)
	case "tasks/pushNotificationConfig/set":
		h.ops.TasksPushNotificationConfigSet(ctx, request, response)
	case "tasks/pushNotificationConfig/get":
		h.ops.TasksPushNotificationConfigGet(ctx, request, response)
	case "tasks/pushNotificationConfig/list":
		h.ops.TasksPushNotificationConfigList(ctx, request, response)
	case "tasks/pushNotificationConfig/delete":
		h.ops.TasksPushNotificationConfigDelete(ctx, request, response)
	case "agent/getAuthenticatedExtendedCard":
		h.ops.AgentGetCard(ctx, request, response)
	default:
		response.Error = jsonrpc.NewMethodNotFound("method not found", nil)
	}
}

func (h *a2aHandler) OnNotification(ctx context.Context, n *jsonrpc.Notification) {
	h.ops.OnNotification(ctx, n)
}

// newA2AHandler constructs a transport-backed handler with Operations.
func newA2AHandler(srv *Server) transport.NewHandler {
	return func(ctx context.Context, t transport.Transport) transport.Handler {
		var ops Operations
		if srv.opsFactory != nil {
			ops = srv.opsFactory(srv, t)
		} else {
			ops = NewOperations(srv, t)
		}
		return &a2aHandler{ops: ops}
	}
}
