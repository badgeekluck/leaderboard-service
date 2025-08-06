package main

import (
	"log"
	"net/http"
	"os"

	"github.com/badgeekluck/leaderboard-service/api"
	"github.com/badgeekluck/leaderboard-service/storage"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env dosyası bulunamadı, varsayılan değerler kullanılacak.")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	serverPort := os.Getenv("PORT")
	if serverPort == "" {
		serverPort = "8080"
	}
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://user:pass@localhost:27017"
	}

	dbStorage, err := storage.NewStorage(redisAddr, mongoURI)

	if err != nil {
		log.Fatalf("Storage başlatılırken hata oluştu: %v", err)
	}

	// 1. WebSocket Hub'ını oluştur
	hub := api.NewHub()
	// 2. Hub'ı kendi iş parçacığında (goroutine) çalıştır
	go hub.Run()

	// 3. API sunucusunu oluştur ve hem storage'ı hem de hub'ı ona ver
	apiServer := api.NewApiServer(dbStorage, hub)

	// 4. Handler'ları kaydet
	http.HandleFunc("/score", apiServer.ScoreHandler)
	http.HandleFunc("/leaderboard", apiServer.LeaderboardHandler)
	// Yeni WebSocket handler'ını kaydet.
	http.HandleFunc("/ws", hub.ServeWs)

	// Test HTML dosyasını sunmak için bir dosya sunucusu ekleyelim
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	log.Printf("Sunucu http://localhost:%s adresinde başlatıldı.", serverPort)
	log.Printf("Test arayüzüne http://localhost:%s/index.html adresinden erişebilirsiniz.", serverPort)

	err = http.ListenAndServe(":"+serverPort, nil)
	if err != nil {
		log.Fatal("Sunucu başlatılırken hata oluştu: ", err)
	}
}
