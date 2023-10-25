package main

import (
    "encoding/json"
    "flag"
    "log"
    "sync"

    "github.com/go-redis/redis/v8" // Importa el paquete Redis
    "github.com/gofiber/contrib/websocket"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "context"
)

type client struct {
    isClosing bool
    mu        sync.Mutex
}

type Message struct {
    ID      string `json:"id"`
    Content string `json:"content"`
    Action  string `json:"action"`
}

var clients = make(map[*websocket.Conn]*client)
var register = make(chan *websocket.Conn)
var broadcast = make(chan Message)
var unregister = make(chan *websocket.Conn)

// Crea un cliente Redis
var redisClient = redis.NewClient(&redis.Options{
    Addr:     "localhost:6379", // Reemplaza esto con la dirección de tu servidor Redis
    Password: "",              // Sin contraseña por defecto
    DB:       0,               // Usar el DB 0 por defecto
})

func runHub() {
    for {
        select {
        case connection := <-register:
            clients[connection] = &client{}
            log.Println("connection registered")

        case message := <-broadcast:
            message.ID = uuid.New().String()
            messageJSON, err := json.Marshal(message)
            if err != nil {
                log.Println("JSON marshaling error:", err)
                continue
            }
            log.Println("message received:", string(messageJSON))
            for connection, c := range clients {
                go func(connection *websocket.Conn, c *client) {
                    c.mu.Lock()
                    defer c.mu.Unlock()
                    if c.isClosing {
                        return
                    }
                    if err := connection.WriteMessage(websocket.TextMessage, messageJSON); err != nil {
                        c.isClosing = true
                        log.Println("write error:", err)

                        connection.WriteMessage(websocket.CloseMessage, []byte{})
                        connection.Close()
                        unregister <- connection
                    }
                }(connection, c)
            }

            // Almacena el mensaje en Redis
            go func(message Message) {
                ctx := context.Background()
                messageBytes, _ := json.Marshal(message)
                err := redisClient.LPush(ctx, "chat_messages", string(messageBytes)).Err()
                if err != nil {
                    log.Println("Redis error:", err)
                }
            }(message)

        case connection := <-unregister:
            delete(clients, connection)
            log.Println("connection unregistered")
        }
    }
}

func main() {
    app := fiber.New()

    app.Static("/", "./home.html")

    app.Use(func(c *fiber.Ctx) error {
        if websocket.IsWebSocketUpgrade(c) {
            return c.Next()
        }
        return c.SendStatus(fiber.StatusUpgradeRequired)
    })

    go runHub()

    app.Get("/ws", websocket.New(func(c *websocket.Conn) {
        defer func() {
            unregister <- c
            c.Close()
        }()

        register <- c

        // Recupera los mensajes anteriores desde Redis y envíalos al cliente recién conectado
        ctx := context.Background()
        messages, err := redisClient.LRange(ctx, "chat_messages", 0, -1).Result()
        if err != nil {
            log.Println("Redis error:", err)
        }
        for _, msgStr := range messages {
            var msg Message
            if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
                log.Println("JSON unmarshaling error:", err)
                continue
            }
            c.WriteJSON(msg)
        }

        for {
            messageType, message, err := c.ReadMessage()
            if err != nil {
                if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                    log.Println("read error:", err)
                }
                return
            }

            if messageType == websocket.TextMessage {
                var msg Message
                if err := json.Unmarshal(message, &msg); err != nil {
                    log.Println("JSON unmarshaling error:", err)
                    continue
                }
                broadcast <- msg
            } else {
                log.Println("websocket message received of type", messageType)
            }
        }
    }))

    addr := flag.String("addr", ":8080", "http service address")
    flag.Parse()
    log.Fatal(app.Listen(*addr))
}
