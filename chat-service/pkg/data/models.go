package data

import (
	"context"
	"fmt"
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
	Type      string    `bson:"type" json:"type"`
	RoomID    string    `bson:"room_id" json:"room_id"`
	SenderID  int       `bson:"sender_id" json:"sender_id"`
	Message   string    `bson:"message" json:"message"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type ChatRoom struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
	// 추가적으로 각 사용자의 마지막 확인 메시지 ID를 추적하기 위한 필드를 고려할 수 있음
	UserLastRead map[string]time.Time `bson:"user_last_read" json:"user_last_read"`
}

// 채팅 메시지 삽입
func (c *Chat) Insert(entry Chat) error {
	collection := client.Database("chat_db").Collection("messages")

	_, err := collection.InsertOne(context.TODO(), Chat{
		Type:      entry.Type,
		RoomID:    entry.RoomID,
		SenderID:  entry.SenderID,
		Message:   entry.Message,
		CreatedAt: entry.CreatedAt,
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

// 채팅 목록 조회 (페이지네이션 포함)
func (c *Chat) GetByRoomIDWithPagination(roomID string, pageNumber int, pageSize int) ([]*Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	opts := options.Find()
	opts.SetSort(bson.D{{"created_at", -1}})         // 최신 순으로 정렬
	opts.SetSkip(int64((pageNumber - 1) * pageSize)) // 페이지에 맞는 메시지 건너뛰기
	opts.SetLimit(int64(pageSize))                   // 페이지당 메시지 수 제한

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
	room.ModifiedAt = time.Now()
	room.UserLastRead = make(map[string]time.Time) // 각 사용자의 마지막 읽은 메시지 ID를 저장하기 위한 맵 (필요 시)

	_, err := collection.InsertOne(ctx, room)
	if err != nil {
		log.Println("Error inserting new room:", err)
		return err
	}

	return nil
}

// 채팅방 정보 업데이트
func (c *ChatRoom) ConfirmRoom(roomID string, userID string, currentTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	// 필터: 해당 Room ID를 가진 문서
	filter := bson.M{"id": roomID}

	// 업데이트 내용: UserLastRead 맵의 해당 사용자에 대한 시간 업데이트
	update := bson.M{
		"$set": bson.M{
			fmt.Sprintf("user_last_read.%s", userID): currentTime,
			"modified_at":                            currentTime,
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

func (c *ChatRoom) UpdateUserLastReadBatch(roomID string, userLastRead map[string]time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("rooms")

	// Update 문서 생성
	updateFields := bson.M{}
	for userID, lastReadTime := range userLastRead {
		updateFields[fmt.Sprintf("user_last_read.%s", userID)] = lastReadTime
	}

	update := bson.M{
		"$set": updateFields,
	}

	// MongoDB UpdateOne 실행
	_, err := collection.UpdateOne(ctx, bson.M{"id": roomID}, update)
	if err != nil {
		log.Printf("Failed to update UserLastRead for room %s: %v", roomID, err)
		return err
	}

	log.Printf("UserLastRead updated for room %s: %+v", roomID, userLastRead)
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

// 특정 유저가 특정 채팅방 내에서 읽지 않은 메시지 개수 반환
func (c *Chat) GetUnreadMessageCount(roomID string, userID string, lastReadTime time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := client.Database("chat_db").Collection("messages")

	// Filter messages created after the user's last read time
	filter := bson.M{
		"room_id":    roomID,
		"created_at": bson.M{"$gt": lastReadTime},
	}

	// Count the documents matching the filter
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		log.Printf("Error counting unread messages for room %s and user %s: %v", roomID, userID, err)
		return 0, err
	}

	return int(count), nil
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
