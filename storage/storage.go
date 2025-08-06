package storage

import (
	"context"
)

// Storage, tüm veritabanı yöneticilerini bir arada tutan ana yapıdır
type Storage struct {
	mongo *MongoStore
	redis *RedisStore
}

// PlayerScore, bir oyuncunun skorunu ve adını temsil eden veri yapısıdır
type PlayerScore struct {
	PlayerID   string  `json:"playerId"`
	PlayerName string  `json:"playerName"`
	Score      float64 `json:"score"`
}

// NewStorage, tüm veritabanı bağlantılarını başlatır ve ana Storage yapısını döndürür
func NewStorage(redisAddr, mongoURI string) (*Storage, error) {
	ctx := context.Background()

	mongoStore, err := NewMongoStore(ctx, mongoURI)
	if err != nil {
		return nil, err
	}

	redisStore, err := NewRedisStore(ctx, redisAddr)
	if err != nil {
		return nil, err
	}

	return &Storage{mongo: mongoStore, redis: redisStore}, nil
}

// AddScore, gelen skoru ilgili veritabanlarına kaydeder
func (s *Storage) AddScore(playerID string, score int) error {
	// Oyuncu profilini MongoDB'de oluştur/kontrol et
	go s.mongo.CreatePlayerIfNotExists(playerID)
	// Skoru Redis'e ekle.
	return s.redis.AddScore(playerID, score)
}

// GetLeaderboard, Redis'ten sıralamayı alır, MongoDB'den oyuncu isimlerini ekler ve birleştirilmiş sonucu döndürür
func (s *Storage) GetLeaderboard() ([]PlayerScore, error) {
	// 1. Redis'ten ID ve skorları al.
	redisScores, err := s.redis.GetLeaderboard()
	if err != nil {
		return nil, err
	}

	if len(redisScores) == 0 {
		return []PlayerScore{}, nil
	}

	// 2. Oyuncu ID'lerini bir listede topla
	playerIDs := make([]string, len(redisScores))
	for i, member := range redisScores {
		playerIDs[i] = member.Member.(string)
	}

	// 3. MongoDB'den bu ID'lere ait isimleri al
	nameMap, err := s.mongo.GetPlayerNames(playerIDs)
	if err != nil {
		return nil, err
	}

	// 4. Sonuçları birleştir.
	playerScores := make([]PlayerScore, len(redisScores))
	for i, member := range redisScores {
		playerName := nameMap[member.Member.(string)]
		if playerName == "" {
			// Eğer MongoDB'de isim henüz yoksa, geçici bir isim kullan.
			playerName = "New Player"
		}
		playerScores[i] = PlayerScore{
			PlayerID:   member.Member.(string),
			PlayerName: playerName,
			Score:      member.Score,
		}
	}

	return playerScores, nil
}
