package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

// singleton
var notifier *WebSocketNotifier

type WebSocketConnectionHandler struct {
	tokenService *auth.TokenService
	notifier     *WebSocketNotifier
}

func NewWebSocketConnectionHandler(tokenService *auth.TokenService) *WebSocketConnectionHandler {
	return &WebSocketConnectionHandler{
		tokenService: tokenService,
		notifier:     NewWebSocketNotifier(),
	}
}

func (h *WebSocketConnectionHandler) WebSocketConnector(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("[ERROR] WebSocket Accept: %s\n", err)
		return
	}
	defer c.CloseNow()

	client := NewWebSocketClient(c)
	h.notifier.register <- client

	ctx := r.Context()

	go client.writePump()

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

	h.notifier.unregister <- client
	c.Close(websocket.StatusNormalClosure, "")
}

type WebSocketClient struct {
	connection *websocket.Conn
	clientID   string
	userID     string
	send       chan []byte
}

func NewWebSocketClient(c *websocket.Conn) *WebSocketClient {
	return &WebSocketClient{
		connection: c,
		clientID:   uuid.NewString(),
		send:       make(chan []byte),
	}
}

func (c *WebSocketClient) writePump() {
	var err error
	var data []byte
	var ctx context.Context
	var cancel context.CancelFunc
	for data = range c.send {
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		err = c.connection.Write(ctx, websocket.MessageText, data)
		if err != nil {
			cancel()
			log.Printf("WebSocket write error: %v", err)
			return // Exit the goroutine on error
		}
		cancel()
	}
}

type NotificationData struct {
	userID string
	data   []byte
}

type WebSocketNotifier struct {
	clients              []*WebSocketClient
	register, unregister chan *WebSocketClient
	notify               chan NotificationData
}

func NewWebSocketNotifier() *WebSocketNotifier {
	if notifier == nil {
		notifier = &WebSocketNotifier{
			clients:    []*WebSocketClient{},
			register:   make(chan *WebSocketClient),
			unregister: make(chan *WebSocketClient),
			notify:     make(chan NotificationData),
		}
		go notifier.Run()
	}

	return notifier
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
		case notification := <-n.notify:
			ind := slices.IndexFunc(n.clients, func(c *WebSocketClient) bool {
				return c.userID == notification.userID
			})
			if ind != -1 {
				n.clients[ind].send <- notification.data
			} else {
				log.Printf("[ERROR] notification user ID not found\n")
			}
		}
	}
}
