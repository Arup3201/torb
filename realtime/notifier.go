package realtime

import (
	"context"
	"log"
	"slices"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type WebSocketClient struct {
	connection       *websocket.Conn
	userID, clientID string
	isAuthenticated  bool
	timer            *time.Timer
	send             chan []byte
}

func NewWebSocketClient(c *websocket.Conn) *WebSocketClient {
	return &WebSocketClient{
		connection: c,
		clientID:   uuid.NewString(),
		send:       make(chan []byte),
	}
}

// TODO: [POTENTIAL BUG] This may cause memory issue. I am not sure if the function
// exits properly when the connection is lost or the client is removed from
// the list of listeners
func (c *WebSocketClient) WritePump() {
	var err error
	var data []byte
	var ctx context.Context
	var cancel context.CancelFunc
	for {
		if c.timer == nil {
			continue
		}

		select {
		case data = <-c.send:
			if !c.isAuthenticated {
				continue
			}

			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			err = c.connection.Write(ctx, websocket.MessageText, data)
			if err != nil {
				cancel()
				log.Printf("WebSocket write error: %v", err)
				return // Exit the goroutine on error
			}
			cancel()
		case <-c.timer.C:
			c.isAuthenticated = false
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			err = c.connection.Write(ctx, websocket.MessageText, []byte(`{"type": "refresh"}`))
			if err != nil {
				cancel()
				log.Printf("WebSocket write error: %v", err)
				return // Exit the goroutine on error
			}
			cancel()
		}
	}
}

func (c *WebSocketClient) Authenticate(userID string) {
	c.userID = userID
	c.isAuthenticated = true
	c.timer = time.NewTimer(auth.ACCESS_TOKEN_DURATION_DEFAULT)
}

type NotificationData struct {
	UserID string
	Data   []byte
}

// singleton
var notifier *WebSocketNotifier

type WebSocketNotifier struct {
	clients              []*WebSocketClient
	register, unregister chan *WebSocketClient
	notify               chan NotificationData
}

func GlobalWebSocketNotifier() *WebSocketNotifier {
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

			if n.clients[ind].timer != nil {
				n.clients[ind].timer.Stop()
			}
			n.clients[ind].connection.Close(websocket.StatusNormalClosure, "")
			n.clients[ind].connection.CloseNow()

			n.clients = append(n.clients[:ind], n.clients[ind+1:]...)
		case notification := <-n.notify:
			ind := slices.IndexFunc(n.clients, func(c *WebSocketClient) bool {
				return c.userID == notification.UserID
			})
			if ind != -1 {
				n.clients[ind].send <- notification.Data
			} else {
				log.Printf("[ERROR] notification user ID not found\n")
			}
		}
	}
}

func (n *WebSocketNotifier) Notify(userID string, data []byte) {
	n.notify <- NotificationData{UserID: userID, Data: data}
}

func (n *WebSocketNotifier) BatchNotify(users []string, data []byte) {
	for _, userID := range users {
		n.notify <- NotificationData{UserID: userID, Data: data}
	}
}

func (n *WebSocketNotifier) Register(client *WebSocketClient) {
	n.register <- client
}

func (n *WebSocketNotifier) Unregister(client *WebSocketClient) {
	n.unregister <- client
}
