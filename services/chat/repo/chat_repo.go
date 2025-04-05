package repo

import (
	"context"
	"errors"
	"fmt"
	"log"
	"solo/pkg/models"
	"solo/pkg/types/commontype"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ChatRepository struct {
	client *mongo.Client
}

func NewChatRepository(client *mongo.Client) (*ChatRepository, error) {
	repo := &ChatRepository{client: client}
	if err := repo.InitDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}
	return repo, nil
}

// InitDatabase 데이터베이스 초기화
func (r *ChatRepository) InitDatabase() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 데이터베이스 생성 (MongoDB는 실제로 데이터가 들어갈 때 생성됨)
	db := r.client.Database("chat_db")

	// 필요한 컬렉션들 생성
	collections := []string{
		"messages",
		"message_readers",
		"rooms",
		"balance_forms",
		"balance_form_votes",
		"balance_form_comments",
		"balance_games",
		"match_histories",
		"room_counter",
	}

	for _, collName := range collections {
		err := db.CreateCollection(ctx, collName)
		if err != nil {
			// 이미 컬렉션이 존재하는 경우 무시
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("Error creating collection %s: %v", collName, err)
				return err
			}
		}
	}

	// 필요한 인덱스 생성
	// messages 컬렉션
	msgCollection := db.Collection("messages")
	_, err := msgCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "room_id", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		log.Printf("Error creating indexes for messages: %v", err)
		return err
	}

	// balance_form_votes 컬렉션 (중복 투표 방지를 위한 복합 인덱스)
	votesCollection := db.Collection("balance_form_votes")
	_, err = votesCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "form_id", Value: 1},
			{Key: "user_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Error creating index for balance_form_votes: %v", err)
		return err
	}

	// 밸런스 게임 초기 데이터 생성
	balanceGames := []models.BalanceGame{
		{
			ID:    primitive.NewObjectID(),
			Title: "연봉과 워라밸",
			Red:   "연봉 2배 받고 주 6일 근무",
			Blue:  "현재 연봉 유지하고 주 4일 근무",
		},
		{
			ID:    primitive.NewObjectID(),
			Title: "이상형의 조건",
			Red:   "외모는 평범하지만 완벽한 성격",
			Blue:  "성격은 평범하지만 완벽한 외모",
		},
		{
			ID:    primitive.NewObjectID(),
			Title: "데이트 비용",
			Red:   "데이트 비용 완벽히 더치페이",
			Blue:  "데이트 비용 번갈아가며 내기",
		},
		{
			ID:    primitive.NewObjectID(),
			Title: "연애 스타일",
			Red:   "매일 연락하는 애정표현 스타일",
			Blue:  "적당한 거리를 두는 독립적인 스타일",
		},
		{
			ID:    primitive.NewObjectID(),
			Title: "기념일",
			Red:   "모든 기념일을 챙기는 로맨티스트",
			Blue:  "큰 기념일만 챙기는 실용주의자",
		},
	}

	// 컬렉션 이름 설정
	collection := r.client.Database("chat_db").Collection("balance_games")

	// 기존 데이터 삭제
	if err := collection.Drop(context.Background()); err != nil {
		log.Printf("Failed to drop balance_games collection: %v", err)
	}

	// 새로운 데이터 삽입
	documents := make([]interface{}, len(balanceGames))
	for i, game := range balanceGames {
		documents[i] = game
	}

	_, err = collection.InsertMany(context.Background(), documents)
	if err != nil {
		log.Printf("Failed to insert balance games: %v", err)
		return err
	}

	return nil
}

// 채팅 메시지 삽입
func (r *ChatRepository) InsertChatMessage(entry models.Chat) (primitive.ObjectID, error) {
	collection := r.client.Database("chat_db").Collection("messages")

	result, err := collection.InsertOne(context.TODO(), entry)
	if err != nil {
		log.Println("Error inserting chat message:", err)
		return primitive.NilObjectID, err
	}

	messageID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		log.Println("Failed to convert InsertedID to ObjectID")
		return primitive.NilObjectID, errors.New("failed to convert InsertedID to ObjectID")
	}

	return messageID, nil
}

// 채팅방 메시지 목록 조회
func (r *ChatRepository) GetChatMessagesByRoomID(roomID string) ([]models.Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, bson.M{"room_id": roomID}, opts)
	if err != nil {
		log.Println("Error fetching chat messages:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Chat
	if err := cursor.All(ctx, &messages); err != nil {
		log.Println("Error decoding chat messages:", err)
		return nil, err
	}

	return messages, nil
}

// 특정 채팅방의 메시지 목록을 페이징 처리하여 조회
func (r *ChatRepository) GetByRoomIDWithPagination(roomID string, pageNumber int, pageSize int) ([]*models.Chat, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")

	// 총 메시지 수 계산
	totalCount, err := collection.CountDocuments(ctx, bson.M{"room_id": roomID})
	if err != nil {
		log.Printf("Error counting messages in room %s: %v", roomID, err)
		return nil, 0, err
	}

	// 메시지 조회
	opts := options.Find()
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}}) // 최신 순으로 정렬
	opts.SetSkip(int64((pageNumber - 1) * pageSize))     // 페이지에 맞는 메시지 건너뛰기
	opts.SetLimit(int64(pageSize))                       // 페이지당 메시지 수 제한

	cursor, err := collection.Find(ctx, bson.M{"room_id": roomID}, opts)
	if err != nil {
		log.Println("Error finding chat messages:", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []*models.Chat
	for cursor.Next(ctx) {
		var item models.Chat
		if err := cursor.Decode(&item); err != nil {
			log.Printf("Error decoding chat message: %v", err)
			return nil, 0, err
		}
		messages = append(messages, &item)
	}

	return messages, totalCount, nil
}

// 특정 방에서 before 시간 이전에 존재하는 읽지 않은 메시지 리스트 조회
func (r *ChatRepository) GetUnreadMessagesBefore(roomID string, before time.Time, userID int) ([]models.Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"room_id": roomID, "created_at": bson.M{"$lt": before}}}},
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
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error finding unread messages for user %d in room %s: %v", userID, roomID, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Chat
	for cursor.Next(ctx) {
		var message models.Chat
		if err := cursor.Decode(&message); err != nil {
			log.Printf("Error decoding message: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// 메시지의 unread_count를 일괄적으로 감소시키는 함수
func (r *ChatRepository) UpdateUnreadCounts(messageIDs []primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")

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

// 특정 방의 최신 메시지 조회
func (r *ChatRepository) GetLastMessageByRoomID(roomID string) (*models.Chat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")

	var lastMessage models.Chat
	err := collection.FindOne(ctx, bson.M{"room_id": roomID}, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&lastMessage)
	if err == mongo.ErrNoDocuments {
		return &models.Chat{
			Message:   "",
			SenderID:  0,
			CreatedAt: time.Time{},
		}, nil
	} else if err != nil {
		log.Println("Finding last chat message error:", err)
		return nil, err
	}

	return &lastMessage, nil
}

// 특정 채팅방 내 채팅 메시지 삭제
func (r *ChatRepository) DeleteChatByRoomID(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("messages")

	filter := bson.M{"room_id": roomID}

	result, err := collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Println("Error deleting chat messages:", err)
		return err
	}

	log.Printf("Deleted %d chat messages for room_id %s", result.DeletedCount, roomID)
	return nil
}

// 채팅 메시지 읽음 처리 삽입
func (r *ChatRepository) InsertChatReader(reader models.ChatReader) error {
	collection := r.client.Database("chat_db").Collection("message_readers")

	_, err := collection.InsertOne(context.TODO(), reader)
	if err != nil {
		log.Println("Error inserting chat reader:", err)
		return err
	}

	return nil
}

// 특정 유저가 특정 방에서 읽지 않은 메시지 개수 조회
func (r *ChatRepository) GetUnreadCountByUserAndRoom(userID int, roomID string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	messagesCollection := r.client.Database("chat_db").Collection("messages")

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

	return 0, nil
}

// 새로운 채팅방 삽입
func (r *ChatRepository) InsertRoom(room *models.ChatRoom) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	_, err := collection.InsertOne(ctx, room)
	if err != nil {
		log.Println("Error inserting new room:", err)
		return err
	}

	return nil
}

// 특정 타입의 방 목록 조회
func (r *ChatRepository) GetAllRoomsOfType(roomType int) ([]models.ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")
	filter := bson.M{"type": roomType}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding rooms of type %d: %v", roomType, err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []models.ChatRoom
	for cursor.Next(ctx) {
		var room models.ChatRoom
		if err := cursor.Decode(&room); err != nil {
			log.Printf("Error decoding room: %v", err)
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// 특정 채팅방 정보 업데이트
func (r *ChatRepository) ConfirmRoom(roomID string, userID string, currentTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	filter := bson.M{"id": roomID}
	update := bson.M{
		"$set": bson.M{
			"modified_at": currentTime,
		},
	}

	_, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Println("Error updating room:", err)
		return err
	}

	return nil
}

// 특정 채팅방 삭제
func (r *ChatRepository) DeleteRoom(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")
	_, err := collection.DeleteOne(ctx, bson.M{"id": roomID})
	if err != nil {
		log.Println("Error deleting room:", err)
		return err
	}

	return nil
}

// 특정 유저가 채팅방을 나가기
func (r *ChatRepository) LeaveRoom(roomID string, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	filter := bson.M{"id": roomID}
	update := bson.M{
		"$pull": bson.M{"user_ids": userID},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error removing user %d from room %s: %v", userID, roomID, err)
		return err
	}

	if result.MatchedCount == 0 {
		log.Printf("No room found with ID %s for user %d to leave", roomID, userID)
		return nil
	}

	log.Printf("User %d left from room %s", userID, roomID)
	return nil
}

// 특정 Room ID로 채팅방 조회
func (r *ChatRepository) GetRoomByID(roomID string) (*models.ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	var room models.ChatRoom
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

// 특정 유저가 포함된 모든 채팅방 조회
func (r *ChatRepository) GetRoomsByUserID(userID int) ([]models.ChatRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")
	cursor, err := collection.Find(ctx, bson.M{"user_ids": userID})
	if err != nil {
		log.Println("Error finding rooms:", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	var rooms []models.ChatRoom
	for cursor.Next(ctx) {
		var room models.ChatRoom
		if err := cursor.Decode(&room); err != nil {
			log.Println("Error decoding room:", err)
			continue
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// 특정 채팅방 내에서 사용자의 게임 정보 조회
func (r *ChatRepository) GetUserGameInfoInRoom(userID int, roomID string) (*models.GamerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	var room models.ChatRoom
	err := collection.FindOne(ctx, bson.M{"id": roomID}).Decode(&room)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("room not found")
		}
		log.Println("Error finding room:", err)
		return nil, err
	}

	for _, gamer := range room.Gamers {
		if gamer.UserID == userID {
			return &gamer, nil
		}
	}

	return nil, errors.New("user not found in the game")
}

// 특정 채팅방의 상태 업데이트
func (r *ChatRepository) UpdateChatRoomStatus(roomID string, status int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("rooms")

	filter := bson.M{"id": roomID}
	update := bson.M{
		"$set": bson.M{
			"status":      status,
			"modified_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating status for room %s: %v", roomID, err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("room not found")
	}

	log.Printf("Successfully updated status for room %s to %d", roomID, status)
	return nil
}

func (r *ChatRepository) GetNextSequence(sequenceName string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	counterCollection := r.client.Database("chat_db").Collection("room_counter")

	filter := bson.M{"_id": sequenceName}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	options := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)

	var result struct {
		Seq int64 `bson:"seq"`
	}
	err := counterCollection.FindOneAndUpdate(ctx, filter, update, options).Decode(&result)
	if err != nil {
		log.Printf("Error generating sequence for %s: %v", sequenceName, err)
		return 0, err
	}

	return result.Seq, nil
}

// 밸런스 게임 폼 조회
func (r *ChatRepository) GetBalanceFormByID(formID primitive.ObjectID) (*models.BalanceGameForm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. 기본 폼 정보 조회
	var form models.BalanceGameForm
	err := r.client.Database("chat_db").Collection("balance_forms").
		FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Printf("Error finding balance form by ID %s: %v", formID.Hex(), err)
		return nil, err
	}

	// 2. 댓글 조회
	cursor, err := r.client.Database("chat_db").Collection("balance_form_comments").
		Find(ctx, bson.M{"balance_form_id": formID})
	if err != nil {
		log.Printf("Error finding comments for balance form %s: %v", formID.Hex(), err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 댓글 디코딩
	var comments []models.BalanceFormComment
	if err = cursor.All(ctx, &comments); err != nil {
		log.Printf("Error decoding comments for balance form %s: %v", formID.Hex(), err)
		return nil, err
	}
	form.Comments = comments

	return &form, nil
}

// 밸런스 게임 폼 삽입
func (r *ChatRepository) InsertBalanceForm(form *models.BalanceGameForm) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_forms")

	// 초기 투표 수는 0으로 설정
	form.Votes = models.Votes{
		RedCount:  0,
		BlueCount: 0,
	}

	result, err := collection.InsertOne(ctx, form)
	if err != nil {
		log.Printf("Error inserting balance form: %v", err)
		return primitive.NilObjectID, err
	}

	formID, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("failed to convert InsertedID to ObjectID")
	}

	return formID, nil
}

// 댓글 추가
func (r *ChatRepository) AddBalanceFormComment(formID primitive.ObjectID, comment *models.BalanceFormComment) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_form_comments")

	// 댓글에 폼 ID 추가
	commentDoc := bson.M{
		"balance_form_id": formID,
		"sender_id":       comment.SenderID,
		"message":         comment.Message,
		"created_at":      time.Now(),
	}

	_, err := collection.InsertOne(ctx, commentDoc)
	if err != nil {
		log.Printf("Error adding comment to form %s: %v", formID.Hex(), err)
		return err
	}

	return nil
}

// 밸런스 게임 폼의 댓글 페이징 조회
func (r *ChatRepository) GetBalanceFormComments(formID primitive.ObjectID, page int, pageSize int) ([]models.BalanceFormComment, int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_form_comments")

	// 총 댓글 수 계산
	totalCount, err := collection.CountDocuments(ctx, bson.M{"balance_form_id": formID})
	if err != nil {
		log.Printf("Error counting comments for form %s: %v", formID.Hex(), err)
		return nil, 0, err
	}

	// 댓글 조회 (최신순)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))

	cursor, err := collection.Find(ctx, bson.M{"balance_form_id": formID}, opts)
	if err != nil {
		log.Printf("Error finding comments for form %s: %v", formID.Hex(), err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var comments []models.BalanceFormComment
	if err = cursor.All(ctx, &comments); err != nil {
		log.Printf("Error decoding comments: %v", err)
		return nil, 0, err
	}

	return comments, totalCount, nil
}

// 사용자의 투표 여부 확인
func (r *ChatRepository) GetUserVote(formID primitive.ObjectID, userID int) (*models.BalanceFormVote, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_form_votes")

	var vote models.BalanceFormVote
	err := collection.FindOne(ctx, bson.M{
		"form_id": formID,
		"user_id": userID,
	}).Decode(&vote)

	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		log.Printf("Error finding vote for user %d in form %s: %v", userID, formID.Hex(), err)
		return nil, err
	}

	return &vote, nil
}

func (r *ChatRepository) GetRoomIdByBalanceFormID(formID primitive.ObjectID) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_forms")

	var form models.BalanceGameForm
	err := collection.FindOne(ctx, bson.M{"_id": formID}).Decode(&form)
	if err != nil {
		log.Printf("Error finding balance form by ID %s: %v", formID.Hex(), err)
		return "", err
	}

	return form.RoomID, nil
}

// 투표 기록 삽입
func (r *ChatRepository) InsertBalanceFormVote(vote *models.BalanceFormVote) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 트랜잭션 시작
	session, err := r.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) error {
		// 1. 이미 투표했는지 확인
		existingVote, err := r.GetUserVote(vote.FormID, vote.UserID)
		if err != nil {
			return err
		}
		if existingVote != nil {
			return errors.New("user already voted")
		}

		// 2. 투표 기록 저장
		voteRecordsCollection := r.client.Database("chat_db").Collection("balance_form_votes")
		vote.CreatedAt = time.Now() // 생성 시간 설정
		_, err = voteRecordsCollection.InsertOne(sessCtx, vote)
		if err != nil {
			log.Printf("Error inserting vote record: %v", err)
			return err
		}

		// 3. 투표 수 증가
		votesCollection := r.client.Database("chat_db").Collection("balance_forms")
		updateField := "votes.blue_cnt"
		if vote.Choiced == commontype.BalanceFormVoteRed {
			updateField = "votes.red_cnt"
		}

		_, err = votesCollection.UpdateOne(sessCtx,
			bson.M{"_id": vote.FormID},
			bson.M{"$inc": bson.M{updateField: 1}})
		if err != nil {
			log.Printf("Error updating vote count: %v", err)
			return err
		}

		return nil
	}

	return mongo.WithSession(ctx, session, callback)
}

// 투표 취소
func (r *ChatRepository) CancelVote(formID primitive.ObjectID, userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 트랜잭션 시작
	session, err := r.client.StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	callback := func(sessCtx mongo.SessionContext) error {
		// 1. 기존 투표 확인
		existingVote, err := r.GetUserVote(formID, userID)
		if err != nil {
			return err
		}
		if existingVote == nil {
			return errors.New("no vote found to cancel")
		}

		// 2. 투표 수 감소
		votesCollection := r.client.Database("chat_db").Collection("balance_forms")
		updateField := "votes.blue_cnt"
		if existingVote.Choiced == commontype.BalanceFormVoteRed {
			updateField = "votes.red_cnt"
		}

		_, err = votesCollection.UpdateOne(sessCtx,
			bson.M{"_id": formID},
			bson.M{"$inc": bson.M{updateField: -1}})
		if err != nil {
			log.Printf("Error decreasing vote count: %v", err)
			return err
		}

		// 3. 투표 기록 삭제
		voteRecordsCollection := r.client.Database("chat_db").Collection("balance_form_votes")
		_, err = voteRecordsCollection.DeleteOne(sessCtx, bson.M{
			"form_id": formID,
			"user_id": userID,
		})
		if err != nil {
			log.Printf("Error deleting vote record: %v", err)
			return err
		}

		return nil
	}

	return mongo.WithSession(ctx, session, callback)
}

func (r *ChatRepository) SaveMatchHistory(matchHistory models.MatchHistory) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("match_histories")

	_, err := collection.InsertOne(ctx, matchHistory)
	if err != nil {
		log.Printf("Error inserting match history: %v", err)
		return err
	}

	return nil
}

func (r *ChatRepository) UpdateMatchHistoryBalanceResult(roomSeq int, balanceResult models.BalanceGameResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("match_histories")

	filter := bson.M{"room_seq": roomSeq}
	update := bson.M{
		"$push": bson.M{
			"balance_results": balanceResult,
		},
	}

	updateResult, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating balance result for room seq %d: %v", roomSeq, err)
		return err
	}

	if updateResult.MatchedCount == 0 {
		return errors.New("match history not found")
	}

	return nil
}

func (r *ChatRepository) UpdateMatchHistoryFinalMatch(roomSeq int, finalMatch []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("match_histories")

	filter := bson.M{"room_seq": roomSeq}
	update := bson.M{
		"$set": bson.M{
			"final_match": finalMatch,
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating final match for room seq %d: %v", roomSeq, err)
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("match history not found")
	}

	return nil
}

// GetRandomBalanceGameForm은 balance_games 컬렉션에서 랜덤한 밸런스 게임을 하나 가져옵니다
func (r *ChatRepository) GetRandomBalanceGameForm() (*models.BalanceGame, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_games")

	// $sample을 사용하여 랜덤하게 하나의 문서를 선택
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Printf("Error getting random balance game: %v", err)
		return nil, err
	}
	defer cursor.Close(ctx)

	// 결과 확인
	if !cursor.Next(ctx) {
		log.Printf("No balance games found in collection")
		return nil, errors.New("no balance games available")
	}

	var game models.BalanceGame
	if err := cursor.Decode(&game); err != nil {
		log.Printf("Error decoding balance game: %v", err)
		return nil, err
	}

	return &game, nil
}

// GetBalanceFormsByRoomID returns all balance forms for a given room
func (r *ChatRepository) GetBalanceFormsByRoomID(roomID string) ([]models.BalanceGameForm, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_forms")
	cursor, err := collection.Find(ctx, bson.M{"room_id": roomID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var forms []models.BalanceGameForm
	if err = cursor.All(ctx, &forms); err != nil {
		return nil, err
	}

	return forms, nil
}

// DeleteBalanceFormVotes deletes all votes for a balance form
func (r *ChatRepository) DeleteBalanceFormVotes(formID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_form_votes")
	_, err := collection.DeleteMany(ctx, bson.M{"form_id": formID})
	return err
}

// DeleteBalanceFormComments deletes all comments for a balance form
func (r *ChatRepository) DeleteBalanceFormComments(formID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_form_comments")
	_, err := collection.DeleteMany(ctx, bson.M{"form_id": formID})
	return err
}

// DeleteBalanceFormsByRoomID deletes all balance forms for a room
func (r *ChatRepository) DeleteBalanceFormsByRoomID(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("balance_forms")
	_, err := collection.DeleteMany(ctx, bson.M{"room_id": roomID})
	return err
}

// DeleteMessageReaders deletes all message readers for a room
func (r *ChatRepository) DeleteMessageReaders(roomID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := r.client.Database("chat_db").Collection("message_readers")
	_, err := collection.DeleteMany(ctx, bson.M{"room_id": roomID})
	return err
}
