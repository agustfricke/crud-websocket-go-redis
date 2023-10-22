package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"

	"github.com/agustfricke/crud-websockets-go-redis/types"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func DeleteMessage(c *fiber.Ctx) error {
  return nil
}

func EditMessage(c *fiber.Ctx) error {
  return nil
}

func CreateMessage(c *fiber.Ctx) error {
    file, err := c.FormFile("upload")
    if err != nil {
        return err
    }

    id := uuid.New()
    ext := filepath.Ext(file.Filename)
    newFilename := fmt.Sprintf("%s%s", id, ext)
    c.SaveFile(file, fmt.Sprintf("public/uploads/%s", newFilename))
    fullPath := fmt.Sprintf("uploads/%s", newFilename)
    fmt.Printf("Archivo guardado en: %s\n", fullPath)

    text := c.FormValue("text")
    title := c.FormValue("title")

    fmt.Printf("Text: %s\n", text)
    fmt.Printf("Title: %s\n", title)

    // Crear una instancia de Message con FullFilePath
    msg := types.Message{
        Text:         text,
        Title:        title,
        File:         fullPath,
    }

    // Enviar el mensaje a través del WebSocket
    types.Broadcast <- msg

    return c.SendStatus(fiber.StatusOK)
}

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
            var msg types.Message 
            if err := json.Unmarshal(message, &msg); err != nil {
                log.Println("json unmarshaling error:", err)
                continue
            }

            log.Printf("WebSocket message received. The file is in: %s", msg.File)

            types.Broadcast <- msg
        } else {
            log.Println("WebSocket message received of type", messageType)
        }
    }
  }


func HomePage(c *fiber.Ctx) error {
	return c.Render("home", fiber.Map{})
}
