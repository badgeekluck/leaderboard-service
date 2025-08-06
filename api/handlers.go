package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/badgeekluck/leaderboard-service/storage"
)

// ApiServer, API handler'larının bağımlılıklarını tutar
type ApiServer struct {
	storage *storage.Storage
	hub     *Hub
}

// ScorePayload, gelen skor verisi için kullanılan yapıdır
type ScorePayload struct {
	PlayerID string `json:"playerId"`
	Score    int    `json:"score"`
}

// NewApiServer, yeni bir ApiServer oluşturur
func NewApiServer(s *storage.Storage, h *Hub) *ApiServer {
	return &ApiServer{storage: s, hub: h}
}

// ScoreHandler, yeni skor gönderme isteklerini yönetir.
func (s *ApiServer) ScoreHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Sadece POST metodu kabul edilir.", http.StatusMethodNotAllowed)
		return
	}

	var payload ScorePayload
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if err != nil {
		http.Error(w, "Geçersiz istek.", http.StatusBadRequest)
		return
	}

	err = s.storage.AddScore(payload.PlayerID, payload.Score)
	if err != nil {
		log.Printf("Storage'a skor eklenirken hata oluştu: %v", err)
		http.Error(w, "Sunucu hatası.", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Yeni skor işlendi -> Oyuncu ID: %s, Skor: %d\n", payload.PlayerID, payload.Score)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Skor başarıyla alındı ve sıralamaya eklendi."))

	// Skor eklendikten sonra, yeni liderlik tablosunu al ve tüm istemcilere yayınla
	// Bunu bir goroutine'de çalıştırarak ana isteğin beklemesini engelleniyor
	go s.broadcastLeaderboard()
}

// broadcastLeaderboard, güncel sıralamayı alıp WebSocket hub'ına gönderir
func (s *ApiServer) broadcastLeaderboard() {
	scores, err := s.storage.GetLeaderboard()
	if err != nil {
		log.Printf("Yayın için liderlik tablosu alınırken hata: %v", err)
		return
	}

	message, err := json.Marshal(scores)
	if err != nil {
		log.Printf("Liderlik tablosu JSON'a çevrilirken hata: %v", err)
		return
	}

	s.hub.BroadcastMessage(message)
}

// LeaderboardHandler, sıralamayı getirme isteklerini yönetir
func (s *ApiServer) LeaderboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Sadece GET metodu kabul edilir.", http.StatusMethodNotAllowed)
		return
	}

	scores, err := s.storage.GetLeaderboard()
	if err != nil {
		log.Printf("Storage'dan liderlik tablosu alınırken hata oluştu: %v", err)
		http.Error(w, "Sunucu hatası.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scores)
}
