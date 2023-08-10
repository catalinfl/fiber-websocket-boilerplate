package main

import (
	"fmt"
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type client struct{}

type Message struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

var clients = make(map[*websocket.Conn]client)
var register = make(chan *websocket.Conn)
var broadcast = make(chan Message)
var unregister = make(chan *websocket.Conn)

func runHub() {
	for {
		select {
		case connection := <-register:
			clients[connection] = client{}
		case message := <-broadcast:
			fmt.Println("message received", message)

			for connection := range clients {
				if err := connection.WriteJSON(message); err != nil {
					fmt.Println("Write error", err)

					unregister <- connection
					connection.WriteMessage(websocket.CloseMessage, []byte{})
					connection.Close()
				}
			}

		case connection := <-unregister:
			delete(clients, connection)
			fmt.Println("connection unregistered")
		}
	}
}

func main() {
	app := fiber.New()

	app.Static("/", "./home.html")

	app.Use(func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Next()
		}
		return nil
	})

	go runHub()

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		defer func() {
			unregister <- c
			c.Close()
		}()

		register <- c

		for {
			messageType, message, err := c.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					fmt.Println("Read error", err)
				}
				return
			}

			if messageType == websocket.TextMessage {

				receivedMessage := Message{
					Type:    strings.Split(string(message), " ")[0],
					Message: strings.Split(string(message), " ")[1],
				}

				broadcast <- receivedMessage
			} else {
				fmt.Println("websocket message received of type", messageType)
			}
		}
	}))

	app.Listen(":3000")

}
