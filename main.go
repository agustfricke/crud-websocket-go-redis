package main

import (
	"log"

	"github.com/agustfricke/crud-websockets-go-redis/handlers"
	"github.com/agustfricke/crud-websockets-go-redis/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func runHub() {
	for {
		select {
		case connection := <-types.Register:
			types.Clients[connection] = &types.Client{}
			log.Println("connection registered")

		case message := <-types.Broadcast:
			log.Println("message received:", message)
			for connection, c := range types.Clients {
				go func(connection *websocket.Conn, c *types.Client) { 
					c.Mu.Lock()
					defer c.Mu.Unlock()
					if c.IsClosing {
						return
					}
					if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
						c.IsClosing = true
						log.Println("write error:", err)

						connection.WriteMessage(websocket.CloseMessage, []byte{})
						connection.Close()
						types.Unregister<- connection
					}
				}(connection, c)
			}

		case connection := <-types.Unregister:
			delete(types.Clients, connection)

			log.Println("connection unregistered")
		}
	}
}

func main() {

    engine := html.New("./templates", ".html")

	  app := fiber.New(fiber.Config{
		    Views: engine, 
        ViewsLayout: "layouts/main", 
	  })


    app.Use("/ws", func(c *fiber.Ctx) error {
        if websocket.IsWebSocketUpgrade(c) {
            c.Locals("allowed", true)
            return c.Next()
        }
        return fiber.ErrUpgradeRequired
    })
    go runHub()

	  app.Get("/", handlers.HomePage)
	  app.Get("/ws", websocket.New(handlers.Chat))

    app.Static("/", "./public")
	  app.Listen(":3000")
}
