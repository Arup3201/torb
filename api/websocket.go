package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Arup3201/torb/auth"
	"github.com/Arup3201/torb/realtime"
	"github.com/coder/websocket"
)

type WebSocketConnectionHandler struct {
	tokenService *auth.TokenService
	notifier     *realtime.WebSocketNotifier
}

func NewWebSocketConnectionHandler(tokenService *auth.TokenService) *WebSocketConnectionHandler {
	return &WebSocketConnectionHandler{
		tokenService: tokenService,
		notifier:     realtime.GlobalWebSocketNotifier(),
	}
}

func (h *WebSocketConnectionHandler) WebSocketConnector(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // TODO: Not Secure
	})
	if err != nil {
		log.Printf("[ERROR] WebSocket Accept: %s\n", err)
		return
	}
	defer c.CloseNow()

	client := realtime.NewWebSocketClient(c)
	h.notifier.Register(client)

	ctx := r.Context()

	done := make(chan struct{})
	go client.WritePump(done)

	var message map[string]string
	var data []byte
	var token, userID string
	var ok bool
	for {
		_, data, err = c.Read(ctx)
		if err != nil {
			break
		}

		if err = json.Unmarshal(data, &message); err != nil {
			log.Printf("[ERROR] WebSocket JSON Unmarshal: %s\n", err)
			c.Write(ctx, websocket.MessageText, []byte(`{"type": "error", "error": "JSON syntax error"}`))
			continue
		}
		if message["type"] == "token" {
			token, ok = message["token"]
			if !ok {
				c.Write(ctx, websocket.MessageText, []byte(`{"type": "error", "error": "Token is missing"}`))
				continue
			}

			userID, err = h.tokenService.GetUserID(ctx, token)
			if err != nil {
				c.Write(ctx, websocket.MessageText, []byte(`{"type": "error", "error": "Invalid token"}`))
				continue
			}

			client.Authenticate(userID)
			c.Write(ctx, websocket.MessageText, []byte(`{"type": "ack"}`))
		}
	}

	h.notifier.Unregister(client)
	close(done)
}
