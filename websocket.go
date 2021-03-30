package sha

import (
	"bytes"
	"context"
	"github.com/zzztttkkk/sha/utils"
	"github.com/zzztttkkk/websocket"
	"net/http"
	"sync"
)

type WebSocketOptions struct {
	ReadBufferSize    int
	WriteBufferSize   int
	EnableCompression bool
	SelectSubprotocol func(ctx *RequestCtx) string
}

var defaultWebSocketProtocolOption = WebSocketOptions{
	ReadBufferSize:    2048,
	WriteBufferSize:   2048,
	EnableCompression: true,
}

type _WebSocketProtocol struct {
	opt WebSocketOptions
	hp  *_Http11Protocol
}

func NewWebSocketProtocol(opt *WebSocketOptions) WebSocketProtocol {
	v := &_WebSocketProtocol{}
	if opt == nil {
		opt = &defaultWebSocketProtocolOption
	}
	v.opt = *opt
	return v
}

const (
	websocketStr         = "websocket"
	websocketExtCompress = "permessage-deflate"
	websocketExt         = "permessage-deflate; server_no_context_takeover; client_no_context_takeover"
)

func (p *_WebSocketProtocol) Handshake(ctx *RequestCtx) (string, bool) {
	version, _ := ctx.Request.Header().Get(HeaderSecWebSocketVersion)
	if len(version) != 2 || version[0] != '1' || version[1] != '3' {
		ctx.Response.statusCode = http.StatusBadRequest
		return "", false
	}

	key, _ := ctx.Request.Header().Get(HeaderSecWebSocketKey)
	if len(key) < 1 {
		ctx.Response.statusCode = http.StatusBadRequest
		return "", false
	}

	var subprotocol string
	if p.opt.SelectSubprotocol != nil {
		subprotocol = p.opt.SelectSubprotocol(ctx)
	}

	var compress bool
	if p.opt.EnableCompression {
		for _, hv := range ctx.Response.Header().GetAll(HeaderSecWebSocketExtensions) {
			if bytes.Contains(hv, utils.B(websocketExtCompress)) {
				compress = true
				break
			}
		}
	}

	res := &ctx.Response
	res.statusCode = http.StatusSwitchingProtocols
	res.Header().AppendString(HeaderConnection, upgrade)
	res.Header().AppendString(HeaderUpgrade, websocketStr)
	res.Header().Append(HeaderSecWebSocketAccept, utils.B(websocket.ComputeAcceptKey(utils.S(key))))
	if compress {
		ctx.Request.webSocketShouldDoCompression = true
		res.Header().Append(HeaderSecWebSocketExtensions, utils.B(websocketExt))
	}
	if len(subprotocol) > 0 {
		ctx.Response.header.SetString(HeaderSecWebSocketProtocol, subprotocol)
	}

	if err := sendResponse(ctx.w, &ctx.Response); err != nil {
		return "", false
	}
	return subprotocol, true
}

var websocketWriteBufferPool sync.Pool

func (p *_WebSocketProtocol) Hijack(ctx *RequestCtx, subprotocol string) *websocket.Conn {
	req := &ctx.Request
	ctx.hijacked = true
	return websocket.NewConnExt(
		ctx.conn, subprotocol, true, req.webSocketShouldDoCompression,
		p.opt.ReadBufferSize, p.opt.WriteBufferSize,
		&websocketWriteBufferPool, ctx.r, nil,
	)
}

type WebsocketHandlerFunc func(ctx context.Context, req *Request, conn *websocket.Conn)

type _WebsocketHandler func(ctx *RequestCtx)

func (w _WebsocketHandler) Handle(ctx *RequestCtx) { w(ctx) }

func wshToHandler(wsh WebsocketHandlerFunc) RequestHandler {
	return _WebsocketHandler(func(ctx *RequestCtx) {
		p := ctx.UpgradeProtocol()
		if p != "websocket" {
			ctx.Response.statusCode = StatusBadRequest
			return
		}
		serv, ok := ctx.Value(CtxKeyServer).(*Server)
		if !ok {
			ctx.Response.statusCode = StatusInternalServerError
			return
		}
		wsp := serv.websocketProtocol
		if wsp == nil {
			ctx.Response.statusCode = StatusBadRequest
			return
		}
		subprotocol, ok := wsp.Handshake(ctx)
		if !ok {
			return
		}
		wsh(ctx, &ctx.Request, wsp.Hijack(ctx, subprotocol))
	})
}
