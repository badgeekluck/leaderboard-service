package storage

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoStore, MongoDB istemcisini ve context'i tutar
type MongoStore struct {
	client *mongo.Client
	ctx    context.Context
}

// NewMongoStore, MongoDB'ye bağlanır ve yeni bir MongoStore örneği döndürür
func NewMongoStore(ctx context.Context, mongoURI string) (*MongoStore, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoURI).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("mongodb'ye bağlanılamadı: %w", err)
	}

	if err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "ping", Value: 1}}).Decode(&bson.M{}); err != nil {
		return nil, fmt.Errorf("mongodb ping'lenemedi: %w", err)
	}

	log.Println("MongoDB sunucusuna başarıyla bağlanıldı.")

	// Bu, aynı ID ile birden fazla oyuncu oluşturulmasını engeller
	playersCollection := client.Database("game").Collection("players")
	indexModel := mongo.IndexModel{
		Keys:    bson.M{"playerId": 1}, // 1: Artan sıralama
		Options: options.Index().SetUnique(true),
	}
	_, err = playersCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		// Index zaten varsa hata vermemesi için logla
		log.Printf("MongoDB unique index oluşturulurken uyarı (zaten var olabilir): %v", err)
	} else {
		log.Println("MongoDB 'playerId' için unique index başarıyla oluşturuldu/kontrol edildi.")
	}

	return &MongoStore{client: client, ctx: ctx}, nil
}

// GetPlayerNames, verilen oyuncu ID'lerine karşılık gelen oyuncu isimlerini bir harita olarak döndürür
func (ms *MongoStore) GetPlayerNames(playerIDs []string) (map[string]string, error) {
	coll := ms.client.Database("game").Collection("players")
	filter := bson.M{"playerId": bson.M{"$in": playerIDs}}
	cursor, err := coll.Find(ms.ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ms.ctx)

	nameMap := make(map[string]string)
	for cursor.Next(ms.ctx) {
		var player struct {
			PlayerID string `bson:"playerId"`
			Name     string `bson:"name"`
		}
		if err := cursor.Decode(&player); err == nil {
			nameMap[player.PlayerID] = player.Name
		}
	}
	return nameMap, nil
}

// CreatePlayerIfNotExists, bir oyuncu ID'si alır ve eğer veritabanında yoksa varsayılan bir profil oluşturur
func (ms *MongoStore) CreatePlayerIfNotExists(playerID string) {
	coll := ms.client.Database("game").Collection("players")
	filter := bson.M{"playerId": playerID}
	opts := options.Update().SetUpsert(true)

	// Kullanıcı bunu daha sonra /player/update API'ı ile güncelleyecek
	update := bson.M{
		"$setOnInsert": bson.M{
			"playerId":  playerID,
			"name":      "New Player",
			"createdAt": time.Now(),
		},
	}

	if _, err := coll.UpdateOne(ms.ctx, filter, update, opts); err != nil {
		// Unique index hatası burada yakalanabilir
		// Sadece logla
		log.Printf("Oyuncu oluşturulurken/kontrol edilirken hata: %v", err)
	}
}

// UpdatePlayerName, belirtilen oyuncunun adını günceller
func (ms *MongoStore) UpdatePlayerName(playerID, newName string) error {
	coll := ms.client.Database("game").Collection("players")
	filter := bson.M{"playerId": playerID}
	update := bson.M{
		"$set": bson.M{"name": newName},
	}

	result, err := coll.UpdateOne(ms.ctx, filter, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("oyuncu bulunamadı: %s", playerID)
	}

	return nil
}
