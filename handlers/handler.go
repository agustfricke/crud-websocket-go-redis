package handlers

import (
	"log"

	"github.com/agustfricke/crud-websockets-go-redis/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func Chat(c *websocket.Conn) {
    defer func() {
			types.Unregister <- c
			c.Close()
		}()

		types.Register <- c

		for {
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Println("read error:", err)
				}

				return 
			}

			if messageType == websocket.TextMessage {
				types.Broadcast <- string(message)
			} else {
				log.Println("websocket message received of type", messageType)
			}
		}
	}

func HomePage(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{})
}
