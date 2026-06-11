package api

import (
	"encoding/json"
	"log"
	"net/http"
	"slices"

	"github.com/Arup3201/torb/auth"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

var notifier = WebSocketNotifier{
	clients:    []*WebSocketClient{},
	register:   make(chan *WebSocketClient),
	unregister: make(chan *WebSocketClient),
}

func init() {
	go notifier.Run()
}

type WebSocketConnectionHandler struct {
	tokenService *auth.TokenService
}

func (h *WebSocketConnectionHandler) WebSocketConnector(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket Accept: %s\n", err)
		return
	}
	defer c.CloseNow()

	client := &WebSocketClient{connection: c, clientID: uuid.NewString()}
	notifier.register <- client

	ctx := r.Context()

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

			client.userID = userID
			c.Write(ctx, websocket.MessageText, []byte(`{"type": "ack"}`))
		} else {
			if client.userID == "" {
				c.Write(ctx, websocket.MessageText, []byte(`{"type": "error", "error": "Client is not authenticated"}`))
			}
		}
	}

	notifier.unregister <- client
	c.Close(websocket.StatusNormalClosure, "")
}

type WebSocketClient struct {
	connection *websocket.Conn
	clientID   string
	userID     string
}

type WebSocketNotifier struct {
	clients              []*WebSocketClient
	register, unregister chan *WebSocketClient
}

func (n *WebSocketNotifier) Run() {
	for {
		select {
		case client := <-n.register:
			n.clients = append(n.clients, client)
		case client := <-n.unregister:
			ind := slices.IndexFunc(n.clients, func(c *WebSocketClient) bool {
				return c.clientID == client.clientID
			})

			n.clients[ind].connection.Close(websocket.StatusNormalClosure, "")
			n.clients[ind].connection.CloseNow()

			n.clients = append(n.clients[:ind], n.clients[ind+1:]...)
		}
	}
}
