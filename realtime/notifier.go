package realtime

import (
	"context"
	"log"
	"slices"
	"sync"
	"time"

	"github.com/Arup3201/torb/auth"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type WebSocketClient struct {
	mt               sync.Mutex
	connection       *websocket.Conn
	userID, clientID string
	authTimer        *time.Timer
	isAuthenticated  bool
	timer            *time.Timer
	send             chan []byte
}

func NewWebSocketClient(c *websocket.Conn) *WebSocketClient {
	return &WebSocketClient{
		connection: c,
		clientID:   uuid.NewString(),
		send:       make(chan []byte, 10), // buffered
		authTimer:  time.NewTimer(10 * time.Second),
		timer:      time.NewTimer(auth.ACCESS_TOKEN_DURATION_DEFAULT),
	}
}

func (c *WebSocketClient) WritePump(done chan struct{}) {
	var err error
	var data []byte
	var ctx context.Context
	var cancel context.CancelFunc
	var timerChan <-chan time.Time
	var isAuthenticated bool
	for {
		c.mt.Lock()
		timerChan = c.timer.C
		isAuthenticated = c.isAuthenticated
		c.mt.Unlock()
		select {
		case <-c.authTimer.C:
			c.connection.Close(websocket.StatusAbnormalClosure, "Authentication timeout")
			return
		case data = <-c.send:
			if isAuthenticated {
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
		case <-timerChan:
			c.mt.Lock()
			c.isAuthenticated = false
			c.mt.Unlock()

			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
			err = c.connection.Write(ctx, websocket.MessageText, []byte(`{"type": "refresh"}`))
			if err != nil {
				cancel()
				log.Printf("WebSocket write error: %v", err)
				return // Exit the goroutine on error
			}
			cancel()
		case <-done:
			return
		}
	}
}

func (c *WebSocketClient) Authenticate(userID string) {
	c.userID = userID

	c.mt.Lock()
	c.isAuthenticated = true
	c.timer.Reset(auth.ACCESS_TOKEN_DURATION_DEFAULT)
	c.mt.Unlock()

	c.authTimer.Stop() // no effect if already expired
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

			if ind != -1 {
				if n.clients[ind].timer != nil {
					n.clients[ind].timer.Stop()
					n.clients[ind].authTimer.Stop() // no effect if already expired
				}
				n.clients[ind].connection.Close(websocket.StatusNormalClosure, "")
				n.clients[ind].connection.CloseNow()

				n.clients = append(n.clients[:ind], n.clients[ind+1:]...)
			}
		case notification := <-n.notify:
			ind := slices.IndexFunc(n.clients, func(c *WebSocketClient) bool {
				return c.userID == notification.UserID
			})
			if ind != -1 {
				select {
				case n.clients[ind].send <- notification.Data:
					// Sent successfully
				default:
					log.Printf("[WARN] Send buffer full for user %s, dropping notification\n", notification.UserID)
				}
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
