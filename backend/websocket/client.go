package ws

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	// Send carries outgoing messages along with the WebSocket message type
	Send   chan *OutgoingMessage
	RoomID string
	UserID string
	Username string
}

// OutgoingMessage represents a message to send to the client and the opcode
type OutgoingMessage struct {
	Type int
	Data []byte
}

// pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("message read error: %v", err)
			break
		}
		c.Hub.Broadcast <- &BroadcastMessage{
			RoomID:  c.RoomID,
			UserID:  c.UserID,
			Message: message,
		}
	}
}

func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for msg := range c.Send {
		if msg == nil {
			continue
		}
		err := c.Conn.WriteMessage(msg.Type, msg.Data)
		if err != nil {
			log.Printf("message write error: %v", err)
			break
		}
	}
}
