package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512000
)

type Client struct {
	Hub  *Hub
	Conn *websocket.Conn
	// Send carries outgoing messages along with the WebSocket message type
	Send     chan *OutgoingMessage
	RoomID   string
	UserID   string
	Username string
}

// OutgoingMessage represents a message to send to the client and the opcode
type OutgoingMessage struct {
	Type int
	Data []byte
}

// ReadPump pumps messages from the websocket connection to the hub.
// It handles the "Pong" response to keep the connection alive.
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// 1. Set limits and deadlines to detect dead connections
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		msgCopy := make([]byte, len(message))
		copy(msgCopy, message)

		// Try to parse as JSON to detect message type
		var jsonMsg map[string]interface{}
		msgType := "text" // default
		if err := json.Unmarshal(msgCopy, &jsonMsg); err == nil {
			if t, ok := jsonMsg["type"].(string); ok {
				msgType = t
			}
		}

		c.Hub.Broadcast <- &BroadcastMessage{
			RoomID:      c.RoomID,
			UserID:      c.UserID,
			Message:     msgCopy, // Send the copy, strictly safe for concurrency
			MessageType: msgType,
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
// It sends "Ping" messages periodically to keep the connection alive.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			// 3. Set write deadlines to prevent hanging if the client is slow/dead
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write the actual message
			err := c.Conn.WriteMessage(msg.Type, msg.Data)
			if err != nil {
				log.Printf("message write error: %v", err)
				return
			}

		case <-ticker.C:
			// 4. Send a Ping periodically
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
