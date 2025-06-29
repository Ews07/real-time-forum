package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	UserUUID string
	Send     chan []byte
}

var (
	clients   = make(map[string]*Client) // key = user UUID
	broadcast = make(chan Message)       // channel for incoming messages
)

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
	SentAt  string `json:"sent_at"`
}

func handleMessages() {
	for {
		msg := <-broadcast

		// If receiver is online, send the message directly
		if client, ok := clients[msg.To]; ok {
			data, _ := json.Marshal(msg)
			client.Send <- data
		}
	}
}

func readPump(db *sql.DB, client *Client) {
	defer func() {
		client.Conn.Close()
		delete(clients, client.UserUUID)
	}()

	for {
		var msg Message
		err := client.Conn.ReadJSON(&msg)
		if err != nil {
			log.Println("read error:", err)
			break
		}

		msg.From = client.UserUUID
		msg.SentAt = time.Now().Format(time.RFC3339)

		broadcast <- msg
	}
}

func writePump(client *Client) {
	for {
		msg, ok := <-client.Send
		if !ok {
			return
		}
		client.Conn.WriteMessage(websocket.TextMessage, msg)
	}
}
