package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/baaami/dorandoran/chat-socket-service/pkg/event"
	"github.com/baaami/dorandoran/chat-socket-service/pkg/types"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
}

func (app *Config) HandleChatSocket(c echo.Context) error {
	// 컨텍스트 생성 및 취소 함수 정의
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "WebSocket upgrade failed")
	}

	xUserID := c.Request().Header.Get("X-User-ID")
	userID, err := strconv.Atoi(xUserID)
	if err != nil {
		log.Printf("User ID is not number, xUserID: %s", xUserID)
		return err
	}

	app.RegisterChatClient(conn, userID)
	defer func() {
		app.UnRegisterChatClient(userID)
		conn.Close()
	}()

	// WaitGroup을 사용하여 모든 고루틴이 종료될 때까지 대기
	var wg sync.WaitGroup
	wg.Add(2)

	// Ping-Pong 메커니즘 추가
	go func() {
		ticker := time.NewTicker(2 * time.Second) // 5초마다 Ping 메시지 전송
		defer wg.Done()
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Ping-Pong goroutine exiting due to context cancellation")
				return
			case <-ticker.C:
				// Ping 메시지 생성 및 전송
				pingMessage := types.WebSocketMessage{Kind: types.MessageKindPing, Payload: nil}
				if err := conn.WriteJSON(pingMessage); err != nil {
					log.Printf("Failed to send ping to user %d: %v", userID, err)
					return
				}
				log.Printf("Sent ping to user %d", userID)

				// 2초 안에 pong 수신 확인
				select {
				case <-app.PongChannel:
					log.Printf("Pong received from user %d", userID)
				case <-time.After(2 * time.Second):
					log.Printf("Pong not received within 2 seconds for user %d", userID)
					conn.Close()
					return
				}
			}
		}
	}()

	// 메시지 처리 고루틴
	go func() {
		defer wg.Done()
		app.listenChatEvent(ctx, conn, userID)
	}()

	// 모든 고루틴이 종료될 때까지 대기
	wg.Wait()

	return nil
}

// 메시지 읽기 처리
func (app *Config) listenChatEvent(ctx context.Context, conn *websocket.Conn, userID int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("Context canceled, exiting listenChatEvent for user %d", userID)
			return // 컨텍스트가 취소되면 고루틴 종료
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
					log.Printf("Unexpected WebSocket close error")
				} else {
					log.Printf("WebSocket connection closed by client")
				}
				return
			}

			var wsMsg types.WebSocketMessage
			if err := json.Unmarshal(msg, &wsMsg); err != nil {
				log.Printf("Failed to unmarshal message: %v", err)
				continue
			}

			switch wsMsg.Kind {
			case types.MessageKindPong:
				select {
				case app.PongChannel <- true:
				default:
					log.Println("Pong received but no ping waiting")
				}
			case types.MessageKindMessage:
				app.handleBroadCastMessage(wsMsg.Payload, userID)
			case types.MessageKindJoin:
				app.handleJoinMessage(wsMsg.Payload, userID)
			case types.MessageKindLeave:
				app.handleLeaveMessage(wsMsg.Payload, userID)
				// 게임방 타임아웃
			case types.MessageKindRoomTimeout:
				app.handleRoomTimeoutMessage(wsMsg.Payload, userID)
				// 최종 선택 메시지
			case types.MessageKindFinalChoice:
				app.handleFinalChoiceMessage(wsMsg.Payload, userID)
			default:
				log.Printf("Unknown Message: %s", wsMsg.Kind)
			}
		}
	}
}

func (app *Config) handleBroadCastMessage(payload json.RawMessage, userID int) {
	var broadCastMsg types.ChatMessage
	if err := json.Unmarshal(payload, &broadCastMsg); err != nil {
		log.Printf("Failed to unmarshal broadcast message: %v", err)
		return
	}

	// 활성 사용자 ID 리스트 가져오기
	activeUserIDs, err := app.RedisClient.GetActiveUserIDs(broadCastMsg.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", broadCastMsg.RoomID, err)
		return
	}

	// 비활성 사용자 ID 리스트 가져오기
	inactiveUserIDs, err := app.RedisClient.GetInActiveUserIDs(broadCastMsg.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", broadCastMsg.RoomID, err)
		return
	}

	// 방에 접속해있는 사용자 ID 리스트 가져오기
	joinedUserIDs, err := app.RedisClient.GetJoinedUser(broadCastMsg.RoomID)
	if err != nil {
		log.Printf("Failed to get joined room users for room %s: %v", broadCastMsg.RoomID, err)
		return
	}

	now := time.Now()
	chat := types.Chat{
		MessageId:   primitive.NewObjectID(),
		Type:        types.ChatTypeChat,
		RoomID:      broadCastMsg.RoomID,
		SenderID:    userID,
		Message:     broadCastMsg.Message,
		UnreadCount: broadCastMsg.HeadCnt - len(joinedUserIDs), // 활성 사용자 수를 이용해 UnreadCount 계산
		CreatedAt:   now,
	}

	// Broadcast to the room
	if err := app.BroadcastToRoom(&chat, joinedUserIDs, activeUserIDs, inactiveUserIDs); err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}
}

// TODO: 최종 선택을 시작하기 위한 조건을 명확히 해야한다.
// -> 모든 클라이언트에서 timeout이 발생 OR Backend에서 timeout이 발생한 방이 모든 클라이언트에게 timeout을 받지 못했을 경우 3초간 대기 후 최종 선택을 강제 시작한다.
// TODO: Backend에서 timeout이 발생한 방이 존재하고 n초 지났을 때 최종 선택을 시작하도록 한다.
// TODO: or 모든 클라이언트에서 timeout이 발생하였을 때 최종 선택을 시작하도록 한다.
func (app *Config) BroadcastFinalChoiceStart(roomID string) error {
	message := types.WebSocketMessage{
		Kind: types.MessageKindFinalChoiceStart,
	}

	// Room 사용자에게 브로드캐스트
	if err := app.sendMessageToRoom(roomID, message); err != nil {
		return fmt.Errorf("failed to broadcast final_choice_start to room %s: %v", roomID, err)
	}

	app.RedisClient.ClearRoomTimeout(roomID)

	log.Printf("Broadcasted final_choice_start event to room %s", roomID)

	return nil
}

// TODO: 최종 선택 완료 후 현황을 공개하기 위한 조건을 명확히 해야한다.
// -> 모든 클라이언트에서 최종 선택 메시지 송신 OR Backend에서 최종 선택 시간이 timeout된 후 모든 클라이언트에게 최종 선택을 받지 못했을 경우 5초간 대기 후 최종 선택을 공개한다.
func (app *Config) BroadcastFinalChoices(roomID string) error {
	// Redis에서 선택 결과 조회
	finalChoiceResults, err := app.RedisClient.GetAllChoices(roomID)
	if err != nil {
		return fmt.Errorf("failed to broadcast final choices: %v", err)
	}

	// JSON으로 직렬화
	payload, err := json.Marshal(finalChoiceResults)
	if err != nil {
		return fmt.Errorf("failed to marshal final choices: %v", err)
	}

	// WebSocket 메시지 생성
	message := types.WebSocketMessage{
		Kind:    types.MessageKindFinalChoiceResult,
		Payload: json.RawMessage(payload),
	}

	// Room 사용자에게 브로드캐스트
	if err := app.sendMessageToRoom(roomID, message); err != nil {
		return fmt.Errorf("failed to broadcast choices to room %s: %v", roomID, err)
	}

	log.Printf("Send %s event, value: %v", types.MessageKindFinalChoiceResult, finalChoiceResults)

	var matchMap sync.Map
	coupleSet := make(map[string]bool) // 중복 방지를 위한 map
	var couples []types.Couple

	// 매칭 데이터를 matchMap에 저장
	for _, userChoice := range finalChoiceResults.Choices {
		matchMap.LoadOrStore(userChoice.UserID, userChoice.SelectedUserID)
	}

	// 매칭 결과로 커플 생성
	for _, userChoice := range finalChoiceResults.Choices {
		selectedUserID := userChoice.SelectedUserID
		userID := userChoice.UserID

		// 현재 사용자를 선택한 사용자가 matchMap에 있는지 확인
		if matchedUserID, ok := matchMap.Load(selectedUserID); ok && matchedUserID == userID {
			// 항상 작은 ID가 UserID1, 큰 ID가 UserID2가 되도록 정렬
			user1, user2 := userID, selectedUserID
			if user1 > user2 {
				user1, user2 = user2, user1
			}

			// 커플을 문자열로 변환하여 중복 확인
			coupleKey := fmt.Sprintf("%d-%d", user1, user2)
			if _, exists := coupleSet[coupleKey]; !exists {
				// 중복되지 않은 경우에만 추가
				couples = append(couples, types.Couple{
					UserID1: user1,
					UserID2: user2,
				})
				coupleSet[coupleKey] = true
			}
		}
	}

	// 커플 데이터를 로그로 확인
	log.Printf("Generated couples for room %s: %+v", roomID, couples)

	err = app.createCoupleRoomEvent(couples)
	if err != nil {
		return fmt.Errorf("failed to createCoupleRoomEvents, err: %v", err)
	}

	app.RedisClient.ClearFinalChoiceRoom(roomID)

	log.Printf("Broadcasted final choices to room %s", roomID)
	return nil
}

func (app *Config) createCoupleRoomEvent(couples []types.Couple) error {
	for _, couple := range couples {
		var matchedMales []types.WaitingUser
		var matchedFemales []types.WaitingUser

		user1, err := GetWaitingUserInfo(strconv.Itoa(couple.UserID1))
		if err != nil {
			log.Printf("Failed to GetWaitingUserInfo, err: %v", err.Error())
			continue
		}

		user2, err := GetWaitingUserInfo(strconv.Itoa(couple.UserID2))
		if err != nil {
			log.Printf("Failed to GetWaitingUserInfo, err: %v", err.Error())
			continue
		}

		if user1.Gender == types.MALE {
			matchedMales = append(matchedMales, *user1)
			matchedFemales = append(matchedFemales, *user2)
		} else {
			matchedMales = append(matchedMales, *user2)
			matchedFemales = append(matchedFemales, *user1)
		}

		// 매칭 ID 생성
		matchID := generateMatchID(matchedMales, matchedFemales)

		matchEvent := types.MatchEvent{
			MatchId:      matchID,
			MatchType:    types.MATCH_COUPLE,
			MatchedUsers: append(matchedMales, matchedFemales...),
		}

		err = app.ChatEmitter.PublishMatchEvent(matchEvent)
		if err != nil {
			log.Printf("Failed to PublishMatchEvent, err: %v", err.Error())
		}
	}
	return nil
}

// generateMatchID creates a unique match ID based on datetime and user IDs
func generateMatchID(males, females []types.WaitingUser) string {
	timestamp := time.Now().Format("20060102150405")
	var userIDs []string
	for _, user := range append(males, females...) {
		userIDs = append(userIDs, strconv.Itoa(user.ID))
	}
	return fmt.Sprintf("%s_%s", timestamp, joinIDs(userIDs))
}

func joinIDs(ids []string) string {
	return strings.Join(ids, "_")
}

func (app *Config) BroadcastToRoom(chatMsg *types.Chat, joinedUserIDs, activeUserIds, inactiveUserIds []int) error {
	chatEvent := types.ChatEvent{
		MessageId:       chatMsg.MessageId,
		Type:            chatMsg.Type,
		RoomID:          chatMsg.RoomID,
		SenderID:        chatMsg.SenderID,
		Message:         chatMsg.Message,
		UnreadCount:     chatMsg.UnreadCount,
		InactiveUserIds: inactiveUserIds,
		ReaderIds:       joinedUserIDs,
		CreatedAt:       chatMsg.CreatedAt,
	}

	// RabbitMQ에 메시지 푸시
	log.Printf("Pushing chat message to RabbitMQ, room: %s, active: %v", chatMsg.RoomID, activeUserIds)
	if err := app.ChatEmitter.PushChatToQueue(chatEvent); err != nil {
		log.Printf("Failed to push chat event to queue, chatMsg: %v, err: %v", chatMsg, err)
		return err
	}

	return nil
}

func (app *Config) sendMessageToRoom(roomID string, message types.WebSocketMessage) error {
	activeUserIDs, err := app.RedisClient.GetActiveUserIDs(roomID)
	if err != nil {
		log.Printf("Failed to get active user id list, err: %s", err.Error())
		return err
	}

	for _, activeUserID := range activeUserIDs {
		if client, ok := app.ChatClients.Load(activeUserID); ok {
			log.Printf("Send Realtime Chat Socket to id: %d, kind: %s", activeUserID, message.Kind)
			if !app.sendMessageToClient(client.(*Client), message) {
				log.Printf("Failed to send message to user %v in room %s", activeUserID, roomID)
			}
		}
	}

	return nil
}

func (app *Config) sendMessageToUser(userID int, message types.WebSocketMessage) error {

	if client, ok := app.ChatClients.Load(userID); ok {
		log.Printf("Send Realtime Chat Socket to id: %d, kind: %s", userID, message.Kind)
		if !app.sendMessageToClient(client.(*Client), message) {
			log.Printf("Failed to send message to user id %v", userID)
		}
	}

	return nil
}

func (app *Config) sendMessageToClient(client *Client, message types.WebSocketMessage) bool {
	select {
	case client.Send <- message:
		return true // 메시지 전송 성공
	case <-time.After(time.Second * 1):
		log.Printf("Timeout while sending message")
		return false // 메시지 전송 실패
	}
}

// Join 메시지 처리
func (app *Config) handleJoinMessage(payload json.RawMessage, userID int) {
	var joinMsg types.JoinRoomMessage
	if err := json.Unmarshal(payload, &joinMsg); err != nil {
		log.Printf("Failed to unmarshal join message: %v, err: %v", payload, err)
		return
	}

	app.JoinRoom(joinMsg.RoomID, userID)
}

// Leave 메시지 처리
func (app *Config) handleLeaveMessage(payload json.RawMessage, userID int) {
	var leaveMsg types.LeaveRoomMessage
	if err := json.Unmarshal(payload, &leaveMsg); err != nil {
		log.Printf("Failed to unmarshal leave message: %v, err: %v", payload, err)
		return
	}

	app.RedisClient.LeaveRoom(leaveMsg.RoomID, userID)
}

// 채팅방 타임아웃 메시지 처리
func (app *Config) handleRoomTimeoutMessage(payload json.RawMessage, userID int) error {
	var roomTimeoutMsg types.RoomTimeoutMessage
	if err := json.Unmarshal(payload, &roomTimeoutMsg); err != nil {
		log.Printf("Failed to unmarshal final choice message: %v, err: %v", payload, err)
		return nil
	}

	err := app.RedisClient.AddTimeoutUser(roomTimeoutMsg.RoomID, userID)
	if err != nil {
		log.Printf("Failed to SaveUserChoice, err: %v", err)
		return nil
	}

	roomTimeoutUserIds, err := app.RedisClient.GetTimeoutUserCount(roomTimeoutMsg.RoomID)
	if err != nil {
		log.Printf("Failed to GetTimeoutUserCount, err: %v", err)
		return nil
	}

	roomTotalUserIds, err := app.RedisClient.GetRoomUserIDs(roomTimeoutMsg.RoomID)
	if err != nil {
		log.Printf("Failed to GetRoomUserIDs, err: %v", err)
		return nil
	}

	if int(roomTimeoutUserIds) == len(roomTotalUserIds) {
		app.BroadcastFinalChoiceStart(roomTimeoutMsg.RoomID)
	}

	return nil
}

// 최종 선택 메시지 처리
func (app *Config) handleFinalChoiceMessage(payload json.RawMessage, userID int) error {
	var finalChoice types.FinalChoiceMessage
	if err := json.Unmarshal(payload, &finalChoice); err != nil {
		log.Printf("Failed to unmarshal final choice message: %v, err: %v", payload, err)
		return nil
	}

	// 최종 선택 완료 이벤트 발생 시
	err := app.RedisClient.SaveUserChoice(finalChoice.RoomID, userID, finalChoice.SelectedUserID)
	if err != nil {
		log.Printf("Failed to SaveUserChoice, err: %v", err)
		return nil
	}

	roomTotalUserIds, err := app.RedisClient.GetRoomUserIDs(finalChoice.RoomID)
	if err != nil {
		log.Printf("Failed to GetRoomUserIDs, err: %v", err)
		return nil
	}

	ok, err := app.RedisClient.IsAllChoicesCompleted(finalChoice.RoomID, int64(len(roomTotalUserIds)))
	if err != nil {
		log.Printf("Failed to IsAllChoicesCompleted, err: %v", err)
		return nil
	}
	if ok {
		app.BroadcastFinalChoices(finalChoice.RoomID)
	}

	return nil
}

func (app *Config) JoinRoom(roomID string, userID int) {
	log.Printf("User %d joined room %s", userID, roomID)

	app.RedisClient.JoinRoom(roomID, userID)

	roomJoinMsg := types.RoomJoinEvent{
		RoomID: roomID,
		UserID: userID,
		JoinAt: time.Now(),
	}

	log.Printf("Pushing room join event to RabbitMQ, roomID: %s, userID: %d, time: %v", roomJoinMsg.RoomID, roomJoinMsg.UserID, roomJoinMsg.JoinAt)

	err := app.ChatEmitter.PushRoomJoinToQueue(types.RoomJoinEvent(roomJoinMsg))
	if err != nil {
		log.Printf("Failed to push room join to queue, roomJoinMsg: %v, err: %v", roomJoinMsg, err)
	}
}

func (app *Config) RegisterChatClient(conn *websocket.Conn, userID int) {
	client := &Client{
		Conn: conn,
		Send: make(chan interface{}, 256),
	}

	// 쓰기 고루틴 시작
	go client.writePump()

	app.ChatClients.Store(userID, client)

	// Redis에 활성 사용자로 등록
	serverID := "unique-server-id" // TODO: 서버의 고유 식별자
	if err := app.RedisClient.RegisterActiveUser(userID, serverID); err != nil {
		log.Printf("Failed to register active user %d in Redis: %v", userID, err)
	} else {
		log.Printf("User %d registered as active on server %s", userID, serverID)
	}

	log.Printf("User %d register chat server", userID)
}

func (app *Config) UnRegisterChatClient(userID int) {
	if clientInterface, ok := app.ChatClients.Load(userID); ok {
		client := clientInterface.(*Client)

		// Send 채널 닫기
		close(client.Send)

		// Channel에서 유저 제거
		app.ChatClients.Delete(userID)

		// Redis에서 활성 사용자 제거
		if err := app.RedisClient.UnregisterActiveUser(userID); err != nil {
			log.Printf("Failed to unregister active user %d in Redis: %v", userID, err)
		} else {
			log.Printf("User %d unregistered as active", userID)
		}

		log.Printf("User %d unregistered chat server", userID)
	}
}

func (c *Client) writePump() {
	defer func() {
		log.Printf("[INFO] writePump for user %v exited", c.Conn.RemoteAddr())
	}()

	for {
		message, ok := <-c.Send
		if !ok {
			// 채널이 닫힌 경우 연결을 닫습니다.
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		// 메시지를 전송합니다.
		if err := c.Conn.WriteJSON(message); err != nil {
			log.Printf("Failed to write message: %v", err)
			return
		}
	}
}

// RabbitMQ 소비자로부터 발생한 이벤트 처리 함수
func (app *Config) SendSocketByChatEvents() {
	for event := range app.EventChannel {
		switch event.Kind {
		case types.MessageKindChatLastest:
			if err := app.handleChatLatestMessage(event.Payload); err != nil {
				log.Printf("Failed to handle chat.latest event: %v", err)
			}
		case types.MessageKindLeave:
			if err := app.handleRoomLeaveMessage(event.Payload); err != nil {
				log.Printf("Failed to handle room.leave event: %v", err)
			}
		case types.MessageKindMessage:
			if err := app.handleChatMessage(event.Payload); err != nil {
				log.Printf("Failed to handle chat event: %v", err)
			}
		case types.MessageKindCoupleMatchSuccess:
			if err := app.handleCoupleMatchSuccessMessage(event.Payload); err != nil {
				log.Printf("Failed to handle chat event: %v", err)
			}
		default:
			log.Printf("Unknown WebSocket event kind: %s", event.Kind)
		}
	}
}

// RabbitMQ 소비자로부터 발생한 chat.latest 이벤트 처리 함수
func (app *Config) handleChatLatestMessage(payload json.RawMessage) error {
	var chatLatest types.ChatLatestEvent
	if err := json.Unmarshal(payload, &chatLatest); err != nil {
		return fmt.Errorf("failed to unmarshal chat.latest payload: %w", err)
	}

	log.Printf("Broadcasting chat.latest event for RoomID: %s", chatLatest.RoomID)

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindChatLastest,
		Payload: payload,
	}

	if err := app.sendMessageToRoom(chatLatest.RoomID, wsMessage); err != nil {
		return fmt.Errorf("failed to broadcast chat.latest for RoomID %s: %w", chatLatest.RoomID, err)
	}

	return nil
}

func (app *Config) handleRoomLeaveMessage(payload json.RawMessage) error {
	var roomLeave event.RoomLeaveEvent
	if err := json.Unmarshal(payload, &roomLeave); err != nil {
		return fmt.Errorf("failed to unmarshal room leave payload: %w", err)
	}

	log.Printf("Broadcasting room leave event, room id: %s, user id: %v", roomLeave.RoomID, roomLeave.LeaveUserID)

	// 활성 사용자 ID 리스트 가져오기
	activeUserIDs, err := app.RedisClient.GetActiveUserIDs(roomLeave.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", roomLeave.RoomID, err)
		return err
	}

	// 비활성 사용자 ID 리스트 가져오기
	inactiveUserIDs, err := app.RedisClient.GetInActiveUserIDs(roomLeave.RoomID)
	if err != nil {
		log.Printf("Failed to get active users for room %s: %v", roomLeave.RoomID, err)
		return err
	}

	// 방에 접속해있는 사용자 ID 리스트 가져오기
	joinedUserIDs, err := app.RedisClient.GetJoinedUser(roomLeave.RoomID)
	if err != nil {
		log.Printf("Failed to get joined room users for room %s: %v", roomLeave.RoomID, err)
		return err
	}

	now := time.Now()
	chat := types.Chat{
		MessageId:   primitive.NewObjectID(),
		Type:        types.ChatTypeLeave,
		RoomID:      roomLeave.RoomID,
		SenderID:    roomLeave.LeaveUserID,
		Message:     "",
		UnreadCount: 0,
		CreatedAt:   now,
	}

	// Broadcast to the room
	if err := app.BroadcastToRoom(&chat, joinedUserIDs, activeUserIDs, inactiveUserIDs); err != nil {
		log.Printf("Failed to broadcast message: %v", err)
	}

	return nil
}

// RabbitMQ 소비자로부터 발생한 chat 이벤트 처리 함수
func (app *Config) handleChatMessage(payload json.RawMessage) error {
	var chatMsg types.ChatEvent
	if err := json.Unmarshal(payload, &chatMsg); err != nil {
		return fmt.Errorf("failed to unmarshal chat payload: %w", err)
	}

	log.Printf("Broadcasting chat event for RoomID: %s", chatMsg.RoomID)

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindMessage,
		Payload: payload,
	}

	if err := app.sendMessageToRoom(chatMsg.RoomID, wsMessage); err != nil {
		return fmt.Errorf("failed to broadcast chat.latest for RoomID %s: %w", chatMsg.RoomID, err)
	}

	return nil
}

func (app *Config) handleCoupleMatchSuccessMessage(payload json.RawMessage) error {
	var chatRoom types.ChatRoom
	if err := json.Unmarshal(payload, &chatRoom); err != nil {
		return fmt.Errorf("failed to unmarshal chat payload: %w", err)
	}

	log.Printf("Broadcasting couple match success event, room id: %s, user id: %v", chatRoom.ID, chatRoom.UserIDs)

	CoupleMatchSuccessMsg := types.CoupleMatchSuccessMessage{
		RoomID: chatRoom.ID,
	}

	data, err := json.Marshal(CoupleMatchSuccessMsg)
	if err != nil {
		log.Printf("Failed to marshal CoupleMatchSuccessMsg data: %v", err)
		return err
	}

	wsMessage := types.WebSocketMessage{
		Kind:    types.MessageKindCoupleMatchSuccess,
		Payload: data,
	}

	for _, userID := range chatRoom.UserIDs {
		err := app.sendMessageToUser(userID, wsMessage)
		if err != nil {
			log.Printf("failed to sendMessageToUser, userID: %s", userID)
			continue
		}
	}

	return nil
}

// [Bridge user] 유저 정보 조회
func GetWaitingUserInfo(userID string) (*types.WaitingUser, error) {
	var user types.User

	// Matching 필터 획득
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, "http://user-service/find", nil)
	if err != nil {
		return nil, err
	}

	// 사용자 ID를 요청의 헤더에 추가
	req.Header.Set("X-User-ID", userID)

	// 요청 실행
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	err = json.Unmarshal(body, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	waitingUser := types.WaitingUser{
		ID:          user.ID,
		Gender:      user.Gender,
		Birth:       user.Birth,
		Address:     types.Address(user.Address),
		CoupleCount: 2,
	}

	return &waitingUser, nil
}
