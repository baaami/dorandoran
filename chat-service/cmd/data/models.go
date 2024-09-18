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
		Chat:     Chat{},
		ChatRoom: ChatRoom{},
	}
}

type Models struct {
	Chat     Chat
	ChatRoom ChatRoom
}

type Chat struct {
	RoomID     string    `bson:"room_id" json:"room_id"`
	SenderID   string    `bson:"sender_id" json:"sender_id"`
	ReceiverID string    `bson:"receiver_id" json:"receiver_id"`
	Message    string    `bson:"message" json:"message"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
}

type ChatRoom struct {
	ID            int       `bson:"id" json:"id"`
	UserAID       int       `bson:"user_a_id" json:"user_a_id"`
	UserBID       int       `bson:"user_b_id" json:"user_b_id"`
	LastConfirmID int       `bson:"last_confirm_id" json:"last_confirm_id"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt    time.Time `bson:"modified_at" json:"modified_at"`
	ConfirmAt     time.Time `bson:"confirm_at" json:"confirm_at"`
}

// 채팅 메시지 삽입
func (c *Chat) Insert(entry Chat) error {
	collection := client.Database("chat_db").Collection("messages")

	_, err := collection.InsertOne(context.TODO(), Chat{
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

// 채팅 목록 조회 (by ChatRoom ID)
func (c *Chat) GetByRoomID(roomID string) ([]*Chat, error) {
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

	var messages []*Chat

	for cursor.Next(ctx) {
		var item Chat

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

// 새로운 채팅방 삽입
func (c *ChatRoom) InsertRoom(room *ChatRoom) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// rooms 컬렉션 선택
	collection := client.Database("chat_db").Collection("rooms")

	// Auto-increment id 계산
	opts := options.FindOne().SetSort(bson.D{{"id", -1}})
	var lastRoom ChatRoom
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(&lastRoom)
	if err != nil && err != mongo.ErrNoDocuments {
		return err
	}
	room.ID = lastRoom.ID + 1
	room.CreatedAt = time.Now()
	room.ModifiedAt = time.Now()
	room.ConfirmAt = time.Now()

	_, err = collection.InsertOne(ctx, room)
	if err != nil {
		log.Println("Error inserting new room:", err)
		return err
	}

	return nil
}

// 채팅방 삭제
func (c *ChatRoom) DeleteRoom(roomID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")
	_, err := collection.DeleteOne(ctx, bson.M{"id": roomID})
	if err != nil {
		log.Println("Error deleting room:", err)
		return err
	}

	return nil
}

// Room ID로 채팅방 조회
func (c *ChatRoom) GetRoomByID(roomID int) (*ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	// MongoDB에서 해당 Room ID로 채팅방을 조회
	var room ChatRoom
	err := collection.FindOne(ctx, bson.M{"id": roomID}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // 해당 Room ID에 해당하는 방이 없을 때
		}
		log.Println("Error finding room by ID:", err)
		return nil, err
	}

	return &room, nil
}

// 특정 유저의 채팅방 목록 조회
func (c *ChatRoom) GetRoomsByUserID(userID int) ([]ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")
	cursor, err := collection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"user_a_id": userID},
			{"user_b_id": userID},
		},
	})
	if err != nil {
		log.Println("Error finding rooms:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []ChatRoom
	for cursor.Next(ctx) {
		var room ChatRoom
		if err := cursor.Decode(&room); err != nil {
			log.Println("Error decoding room:", err)
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}
