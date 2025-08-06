package api

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Upgrader, standart bir HTTP bağlantısını WebSocket bağlantısına çevirir
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Hub, tüm aktif istemcileri ve onlara gönderilecek mesajları yönetir
type Hub struct {
	clients    map[*websocket.Conn]bool // Aktif istemcilerin bir haritası
	broadcast  chan []byte              // Gelen mesajların gönderileceği kanal
	register   chan *websocket.Conn     // Yeni istemci kayıt istekleri için kanal
	unregister chan *websocket.Conn     // İstemci kaldırma istekleri için kanal
	mu         sync.Mutex               // clients haritasına güvenli erişim için Mutex
}

// NewHub, yeni bir Hub oluşturur ve başlatır
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run, Hub'ı çalıştırır. Yeni istemcileri, ayrılanları ve mesajları dinler
// Bu fonksiyonun ayrı bir goroutine'de çalıştırılması gerekir
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Println("Yeni bir istemci bağlandı.")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Close()
				log.Println("Bir istemcinin bağlantısı kesildi.")
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				err := client.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("İstemciye mesaj gönderilirken hata: %v. Bağlantı kaldırılıyor.", err)
					// Hata durumunda istemciyi otomatik olarak kaldır
					client.Close()
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastMessage, dışarıdan bir mesajı hub'ın broadcast kanalına gönderir
func (h *Hub) BroadcastMessage(message []byte) {
	h.broadcast <- message
}

// ServeWs, HTTP isteklerini WebSocket bağlantılarına yükseltir ve hub'a kaydeder
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Yeni istemciyi hub'a kaydet.
	h.register <- conn

	// İstemci bağlantısı koptuğunda unregister kanalına gönder
	// Bu, tarayıcı sekmesi kapatıldığında veya bağlantı koptuğunda tetiklenir
	defer func() { h.unregister <- conn }()

	// Bu döngü, istemciden gelebilecek mesajları dinlemek için vardır
	// Bizim projemizde istemci mesaj göndermeyecek, ama bu yapı standarttır
	// Eğer istemci bağlantısı koparsa, döngü sonlanır ve defer çalışır
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
