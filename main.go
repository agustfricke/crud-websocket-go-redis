package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

type client struct {
  isClosing bool
  mu        sync.Mutex
}

var clients = make(map[*websocket.Conn]*client)
var register = make(chan *websocket.Conn)
var unregister = make(chan *websocket.Conn)

var broadcast = make(chan string)
var deleteMessage = make(chan string)

type messageStore struct {
  sync.Mutex
  client *redis.Client
}

var messageStorage *messageStore

func initializeRedis() (*redis.Client, error) {
  client := redis.NewClient(&redis.Options{
    Addr: "localhost:6379", 
    DB:   0,               
  })
  _, err := client.Ping(client.Context()).Result()
  return client, err
}

func runHub() {
  for {
    select {
    case connection := <-register:
      clients[connection] = &client{}
      log.Println("connection registered")

      messageStorage.Lock()
      messages, err := messageStorage.client.LRange(messageStorage.client.Context(), "messages", 0, -1).Result()
      if err == nil {
        for _, message := range messages {
          connection.WriteMessage(websocket.TextMessage, []byte(message))
        }
      }
      messageStorage.Unlock()

    case message := <-broadcast:
      log.Println("message received:", message)

      messageStorage.Lock()
      messageStorage.client.LPush(messageStorage.client.Context(), "messages", message)
      messageStorage.Unlock()

      for connection, c := range clients {
        go func(connection *websocket.Conn, c *client) {
          c.mu.Lock()
          defer c.mu.Unlock()
          if c.isClosing {
            return
          }
          if err := connection.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
            c.isClosing = true
            log.Println("write error:", err)

            connection.WriteMessage(websocket.CloseMessage, []byte{})
            connection.Close()
            unregister <- connection
          }
        }(connection, c)
      }

    case message := <-deleteMessage:
      log.Println("message to delete:", message)

      messageStorage.Lock()
      if err := messageStorage.client.LRem(messageStorage.client.Context(), "messages", 0, message).Err(); err != nil {
        log.Println("error deleting message:", err)
      }
      messageStorage.Unlock()

      for connection, c := range clients {
        go func(connection *websocket.Conn, c *client) {
          c.mu.Lock()
          defer c.mu.Unlock()
          if c.isClosing {
            return
          }
          if err := connection.WriteMessage(websocket.TextMessage, []byte("Deleted message: "+message)); err != nil {
            c.isClosing = true
            log.Println("error al eliminar write error:", err)

            connection.WriteMessage(websocket.CloseMessage, []byte{})
            connection.Close()
            unregister <- connection
          }
        }(connection, c)
      }
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

  redisClient, err := initializeRedis()
  if err != nil {
    log.Fatal("Failed to connect to Redis:", err)
  }
  messageStorage = &messageStore{
    client: redisClient,
  }

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
          log.Println("read error:", err)
        }
        return
      }

      if messageType == websocket.TextMessage {

        messageStr := string(message)
        log.Println("websocket message received:", messageStr)

        parts := strings.Split(messageStr, " || ")
        data := make(map[string]string)

        for _, part := range parts {
          keyValue := strings.Split(part, ": ")
          if len(keyValue) == 2 {
            key := strings.TrimSpace(keyValue[0])
            value := strings.TrimSpace(keyValue[1])
            data[key] = value
          }
        }

        id := data["id"]
        title := data["title"]
        action := data["action"]

        if (action == "delete") {
          messageToBeDeleted := fmt.Sprintf("id: %s || title: %s || action: normal", id, title)
          deleteMessage <- messageToBeDeleted
          fmt.Println("El mensaje se elimino")
        } else {
          broadcast <- messageStr
        }
      }
    }
  }))

  addr := flag.String("addr", ":8080", "http service address")
  flag.Parse()
  log.Fatal(app.Listen(*addr))
}
