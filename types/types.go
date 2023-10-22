package types

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type Message struct {
    Title       string `json:"title"`
    Text        string `json:"text"`
    File        string `json:"file"`
}

type Client struct {
  IsClosing bool
	Mu        sync.Mutex
}

var Clients = make(map[*websocket.Conn]*Client) 
var Register = make(chan *websocket.Conn)
var Broadcast = make(chan Message) 
var Unregister = make(chan *websocket.Conn)
