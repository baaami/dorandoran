package data

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client

func New(mongo *mongo.Client) Models {
	client = mongo

	return Models{
		Chat:       Chat{},
		ChatRoom:   ChatRoom{},
		ChatReader: ChatReader{},
	}
}

type Models struct {
	Chat       Chat
	ChatRoom   ChatRoom
	ChatReader ChatReader
}

// 채팅 메시지 삽입
func (c *Chat) Insert(entry Chat) (primitive.ObjectID, error) {
	collection := client.Database("chat_db").Collection("messages")

	// Insert the chat message into the collection
	result, err := collection.InsertOne(context.TODO(), Chat{
		MessageId:   entry.MessageId,
		Type:        entry.Type,
		RoomID:      entry.RoomID,
		SenderID:    entry.SenderID,
		Message:     entry.Message,
		UnreadCount: entry.UnreadCount,
		CreatedAt:   entry.CreatedAt,
	})
	if err != nil {
		log.Println("Error inserting chat message:", err)
		return primitive.NilObjectID, err
	}

	// Convert the inserted ID to primitive.ObjectID
	messageID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Println("Failed to convert InsertedID to ObjectID")
		return primitive.NilObjectID, fmt.Errorf("failed to convert InsertedID to ObjectID")
	}

	return messageID, nil
}

// 채팅 메시지 읽은 사용자 삽입
func (cr *ChatReader) Insert(reader ChatReader) error {
	collection := client.Database("chat_db").Collection("message_readers")

	_, err := collection.InsertOne(context.TODO(), reader)
	if err != nil {
		log.Println("Error inserting chat reader:", err)
		return err
	}

	return nil
}

// 해당 room에서 before 시간 이전에 존재한 메시지 리스트
func (c *Chat) GetMessagesBefore(roomID string, before time.Time) ([]Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// 필터 조건: 특정 RoomID 및 CreatedAt < before
	filter := bson.M{
		"room_id":    roomID,
		"created_at": bson.M{"$lt": before},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding messages for RoomID %s: %v", roomID, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []Chat
	for cursor.Next(ctx) {
		var message Chat
		if err := cursor.Decode(&message); err != nil {
			log.Printf("Error decoding message: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// 메시지의 unread_count를 일괄적으로 감소시키는 함수
func (c *Chat) UpdateUnreadCounts(messageIDs []primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// UpdateMany를 사용하여 일괄 감소
	filter := bson.M{"_id": bson.M{"$in": messageIDs}}
	update := bson.M{"$inc": bson.M{"unread_count": -1}}

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		log.Printf("Failed to update unread_count for messages: %v", err)
		return err
	}

	log.Printf("Updated unread_count for %d messages", result.ModifiedCount)
	return nil
}

// 읽지 않은 메시지 개수 획득
func (c *ChatReader) GetUnreadCountByUserAndRoom(userID int, roomID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	messagesCollection := client.Database("chat_db").Collection("messages")

	// Step 1: 찾을 메시지들의 `_id` 추출
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"room_id": roomID}}},
		bson.D{{Key: "$lookup", Value: bson.M{
			"from":         "message_readers",
			"localField":   "_id",
			"foreignField": "message_id",
			"as":           "readers",
		}}},
		bson.D{{Key: "$match", Value: bson.M{
			"$expr": bson.M{
				"$not": bson.M{
					"$in": bson.A{userID, "$readers.user_id"},
				},
			},
		}}},
		bson.D{{Key: "$count", Value: "unread_count"}},
	}

	cursor, err := messagesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error retrieving unread count for user %d in room %s: %v", userID, roomID, err)
		return 0, err
	}
	defer cursor.Close(ctx)

	// Step 2: 결과 처리
	var result struct {
		UnreadCount int `bson:"unread_count"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			log.Printf("Error decoding unread count result: %v", err)
			return 0, err
		}
		return result.UnreadCount, nil
	}

	return 0, nil // 읽지 않은 메시지가 없는 경우 0 반환
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

// 채팅 목록 조회
func (c *Chat) GetByRoomIDWithPagination(roomID string, pageNumber int, pageSize int) ([]*Chat, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// 총 메시지 수 계산
	totalCount, err := collection.CountDocuments(ctx, bson.M{"room_id": roomID})
	if err != nil {
		log.Printf("Error counting messages in room %s: %v", roomID, err)
		return nil, 0, err
	}

	// 메시지 조회
	opts := options.Find()
	opts.SetSort(bson.D{{"created_at", -1}})         // 최신 순으로 정렬
	opts.SetSkip(int64((pageNumber - 1) * pageSize)) // 페이지에 맞는 메시지 건너뛰기
	opts.SetLimit(int64(pageSize))                   // 페이지당 메시지 수 제한

	cursor, err := collection.Find(ctx, bson.M{"room_id": roomID}, opts)
	if err != nil {
		log.Println("Finding chat messages error:", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []*Chat
	for cursor.Next(ctx) {
		var item Chat
		err := cursor.Decode(&item)
		if err != nil {
			log.Printf("Error decoding chat message: %v", err)
			return nil, 0, err
		}
		messages = append(messages, &item)
	}

	return messages, totalCount, nil
}

// 채팅 목록 삭제 (by ChatRoom ID)
func (c *Chat) DeleteChatByRoomID(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// 삭제할 문서의 필터 조건 설정
	filter := bson.M{"room_id": roomID}

	// DeleteMany 메서드를 사용하여 해당 조건의 모든 문서 삭제
	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Println("Error deleting chat messages:", err)
		return err
	}

	log.Printf("Deleted %d chat messages for room_id %s", result.DeletedCount, roomID)

	return nil
}

// 새로운 채팅방 삽입
func (c *ChatRoom) InsertRoom(room *ChatRoom) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	room.CreatedAt = time.Now()
	room.ModifiedAt = room.CreatedAt

	room.FinishChatAt = room.CreatedAt.Add(time.Hour)

	_, err := collection.InsertOne(ctx, room)
	if err != nil {
		log.Println("Error inserting new room:", err)
		return err
	}

	return nil
}

// GetAllRoomsOfType: 특정 타입의 방 목록을 가져오는 함수
func (c *ChatRoom) GetAllRoomsOfType(roomType int) ([]ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	// 필터 조건: Room Type이 roomType에 해당하는 모든 방
	filter := bson.M{"type": roomType}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding rooms of type %d: %v", roomType, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []ChatRoom
	for cursor.Next(ctx) {
		var room ChatRoom
		if err := cursor.Decode(&room); err != nil {
			log.Printf("Error decoding room: %v", err)
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// 채팅방 정보 업데이트
func (c *ChatRoom) ConfirmRoom(roomID string, userID string, currentTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	// 필터: 해당 Room ID를 가진 문서
	filter := bson.M{"id": roomID}

	update := bson.M{
		"$set": bson.M{
			"modified_at": currentTime,
		},
	}

	// 업데이트 실행
	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}

	return nil
}

// 채팅방 삭제
func (c *ChatRoom) DeleteRoom(roomID string) error {
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
func (c *ChatRoom) GetRoomByID(roomID string) (*ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	var room ChatRoom
	err := collection.FindOne(ctx, bson.M{"id": roomID}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Println("Error finding room by ID:", err)
		return nil, err
	}

	return &room, nil
}

// 특정 유저의 채팅방 목록 조회
func (c *ChatRoom) GetRoomsByUserID(userID string) ([]ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")
	cursor, err := collection.Find(ctx, bson.M{
		"users": userID,
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

// 최신 채팅 데이터 조회
func (c *Chat) GetLastMessageByRoomID(roomID string) (*Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// 최신 메시지를 가져오기 위해 내림차순 정렬
	var lastMessage Chat
	err := collection.FindOne(ctx, bson.M{"room_id": roomID}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&lastMessage)
	if err != nil {
		log.Println("Finding last chat message error:", err)
		return nil, err
	}

	return &lastMessage, nil
}
