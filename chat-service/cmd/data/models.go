package data

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func New(mongo *mongo.Client) Models {
	client = mongo

	return Models{
		ChatEntry: ChatEntry{},
	}
}

type Models struct {
	ChatEntry ChatEntry
}

type ChatEntry struct {
	RoomID     string    `bson:"room_id" json:"room_id"`
	SenderID   string    `bson:"sender_id" json:"sender_id"`
	ReceiverID string    `bson:"receiver_id" json:"receiver_id"`
	Message    string    `bson:"message" json:"message"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

func (c *ChatEntry) Insert(entry ChatEntry) error {
	collection := client.Database("chat_db").Collection("messages")

	_, err := collection.InsertOne(context.TODO(), ChatEntry{
		RoomID:     entry.RoomID,
		SenderID:   entry.SenderID,
		ReceiverID: entry.ReceiverID,
		Message:    entry.Message,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		log.Println("Error inserting chat message:", err)
		return err
	}

	return nil
}

func (c *ChatEntry) GetByRoomID(roomID string) ([]*ChatEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	opts := options.Find()
	opts.SetSort(bson.D{{"created_at", -1}})

	cursor, err := collection.Find(context.TODO(), bson.M{"room_id": roomID}, opts)
	if err != nil {
		log.Println("Finding chat messages error:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*ChatEntry

	for cursor.Next(ctx) {
		var item ChatEntry

		err := cursor.Decode(&item)
		if err != nil {
			log.Print("Error decoding chat message:", err)
			return nil, err
		} else {
			messages = append(messages, &item)
		}
	}

	return messages, nil
}
