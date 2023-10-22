package types

import (
	"sync"

	"github.com/gofiber/contrib/websocket"
)

type Message struct {
    Text         string `json:"text"`
    Title        string `json:"title"`
    FullFilePath string `json:"fullFilePath"`
}

type Client struct {
  IsClosing bool
	Mu        sync.Mutex
}

var Clients = make(map[*websocket.Conn]*Client) 
var Register = make(chan *websocket.Conn)
var Broadcast = make(chan string)
var Unregister = make(chan *websocket.Conn)
