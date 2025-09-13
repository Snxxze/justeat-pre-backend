package ws

import (
	"backend/entity"
	"backend/services"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ChatHub คือศูนย์กลางของระบบแชทผ่าน WebSocket
type ChatHub struct {
	clients    map[uint]map[*websocket.Conn]bool // roomID -> set of clients
	broadcast  chan BroadcastMessage
	register   chan Subscription
	unregister chan Subscription
	mu         sync.Mutex
	service    *services.ChatService
}

// Subscription = การสมัครสมาชิกห้อง (1 user ต่อ 1 connection)
type Subscription struct {
	Conn   *websocket.Conn // การเชื่อมต่อ WS ของ client
	RoomID uint
	UserID uint
}

// BroadcastMessage = ข้อความที่จะส่งกระจายให้ทุกคนในห้อง
type BroadcastMessage struct {
	RoomID  uint
	Message *entity.Message
}

// สร้าง ChatHub ใหม่
func NewChatHub(service *services.ChatService) *ChatHub {
	return &ChatHub{
		clients:    make(map[uint]map[*websocket.Conn]bool),
		broadcast:  make(chan BroadcastMessage),
		register:   make(chan Subscription),
		unregister: make(chan Subscription),
		service:    service,
	}
}

// คอยฟัง register/unregister/broadcast ตลอดเวลา
func (h *ChatHub) Run() {
	for {
		select {
			// มี client ใหม่เข้าห้อง
		case sub := <-h.register:
			h.mu.Lock()
			if h.clients[sub.RoomID] == nil {
				h.clients[sub.RoomID] = make(map[*websocket.Conn]bool)
			}
			h.clients[sub.RoomID][sub.Conn] = true
			h.mu.Unlock()

			// มี client ออกจากห้อง
		case sub := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[sub.RoomID][sub.Conn]; ok {
				delete(h.clients[sub.RoomID], sub.Conn)
				sub.Conn.Close()
			}
			h.mu.Unlock()

			// มีข้อความใหม่เข้ามา → กระจายให้ทุกคนในห้อง
		case msg := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients[msg.RoomID] {
				if err := conn.WriteJSON(msg.Message); err != nil {
					log.Printf("ws write error: %v", err)
					conn.Close()
					delete(h.clients[msg.RoomID], conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// WS route: /ws/chat/:roomId
func (h *ChatHub) HandleWebSocket(c *gin.Context) {
    roomIDStr := c.Param("roomId")
    var roomID uint
    fmt.Sscan(roomIDStr, &roomID)

		// --- ดึง userId จาก JWT ที่ middleware ใส่ไว้
    userIDVal, _ := c.Get("userId")
    userID := userIDVal.(uint)

    // --- ตรวจสอบว่าห้องนี้มีจริงไหม
    room, err := h.service.GetRoomByID(roomID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }

		// --- ตรวจสอบสิทธิ์ (เจ้าของ order หรือ rider เท่านั้น)
    ok, err := h.service.CanAccessRoom(userID, room.OrderID)
    if err != nil || !ok {
        c.JSON(http.StatusForbidden, gin.H{"error": "no access"})
        return
    }

		// --- Upgrade HTTP → WebSocket
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("ws upgrade error: %v", err)
        return
    }

		// --- สมัคร client เข้า room
    sub := Subscription{Conn: conn, RoomID: room.ID, UserID: userID}
    h.register <- sub

    go h.listenMessages(sub)
}

// listenMessages = ฟังข้อความใหม่จาก client ทาง WS
func (h *ChatHub) listenMessages(sub Subscription) {
	defer func() { h.unregister <- sub }()

	for {
		_, msgData, err := sub.Conn.ReadMessage()
		if err != nil {
			log.Printf("ws read error: %v", err)
			break
		}

		var payload struct {
			Body          string `json:"body"`
			TypeMessageID uint   `json:"typeMessageId"`
		}
		if err := json.Unmarshal(msgData, &payload); err != nil {
			log.Printf("invalid payload: %v", err)
			continue
		}

		// ใช้ user จาก JWT ไม่ใช่ FE
		msg, err := h.service.SendMessage(sub.RoomID, sub.UserID, payload.TypeMessageID, payload.Body)
		if err != nil {
			log.Printf("save msg error: %v", err)
			continue
		}

		h.broadcast <- BroadcastMessage{RoomID: sub.RoomID, Message: msg}
	}
}


