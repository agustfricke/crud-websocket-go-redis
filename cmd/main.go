package main

import (
	"encoding/json"
	"log"

	"github.com/agustfricke/crud-websockets-go-redis/handlers"
	"github.com/agustfricke/crud-websockets-go-redis/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func RunHub() {
    for {
        select {
        case connection := <- types.Register:
            types.Clients[connection] = &types.Client{}
            log.Println("connection registered")

        case message := <-types.Broadcast:
            log.Println("title received:", message.Title)
            log.Println("message received:", message.Text)
            messageJSON, err := json.Marshal(message)
            if err != nil {
                log.Println("json marshaling error:", err)
                continue
            }
            for connection, c := range types.Clients {
                go func(connection *websocket.Conn, c *types.Client) {
                    c.Mu.Lock()
                    defer c.Mu.Unlock()
                    if c.IsClosing {
                        return
                    }
                    if err := connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
                        c.IsClosing = true
                        log.Println("write error:", err)

                        connection.WriteMessage(websocket.CloseMessage, []byte{})
                        connection.Close()
                        types.Unregister <- connection
                    }
                }(connection, c)
            }

        case connection := <-types.Unregister:
            delete(types.Clients, connection)

            log.Println("connection unregistered")
        }
    }
}


func SetupRoutes(app *fiber.App) {
	  app.Get("/", handlers.HomePage)
	  app.Post("/create", handlers.CreateMessage)
    app.Put("/edit/:id", handlers.EditMessage)
    app.Delete("/delete/:id", handlers.DeleteMessage)

	  app.Get("/ws", websocket.New(handlers.Chat))
}

func main() {

    engine := html.New("./templates", ".html")

	  app := fiber.New(fiber.Config{
		    Views: engine, 
        ViewsLayout: "layouts/main", 
	  })

    app.Static("/", "./public")
    app.Use("/ws", func(c *fiber.Ctx) error {
        if websocket.IsWebSocketUpgrade(c) {
            c.Locals("allowed", true)
            return c.Next()
        }
        return fiber.ErrUpgradeRequired
    })

    go RunHub()

    SetupRoutes(app)

	  app.Listen(":8080")
}
