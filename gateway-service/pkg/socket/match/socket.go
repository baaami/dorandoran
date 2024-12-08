package match

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/baaami/dorandoran/broker/pkg/redis"
	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Kind    string          `json:"kind"`
	Payload json.RawMessage `json:"payload"`
}

type ChatRoom struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []string  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
}

const (
	MessageTypeChat  = "chat"
	MessageTypeMatch = "match"
	MessageTypeRoom  = "room"
)

// Chat Type (Receive)
const (
	MessageStatusChatBroadCast = "broadcast"
)

// Room Type (Receive)
const (
	MessageStatusRoomJoin  = "join"
	MessageStatusRoomLeave = "leave"
)

// Game Type (Receive)
const (
	MessageStatusGameFirstImpressionVote = "first_impression_vote" // 첫인상 투표
	MessageStatusGameSecretChatRequest   = "secret_chat_request"   // 비밀 채팅권 사용
	MessageStatusGameFinalSelection      = "final_selection"       // 최종 선택
)

// Room Type (Push)
const (
	PushMessageStatusRoomInfo = "info"
)

// Match Type (Push)
const (
	PushMessageStatusMatchSuccess = "success"
	PushMessageStatusMatchFailure = "fail"
)

// Game Type (Push)
const (
	PushMessageStatusGameStart                     = "start"
	PushMessageStatusGameIntroduceStart            = "introduce_start"
	PushMessageStatusGameIntroduceTurn             = "introduce_turn"
	PushMessageStatusGameIntroduceEnd              = "introduce_end"
	PushMessageStatusGameFirstImpressionVoteStart  = "first_impression_vote_start"
	PushMessageStatusGameFirstImpressionVoteEnd    = "first_impression_vote_end"
	PushMessageStatusGameFirstImpressionVoteResult = "first_impression_vote_result"
	PushMessageStatusGameMiniGameStart             = "mini_game_start"
	PushMessageStatusGameMiniGameEnd               = "mini_game_end"
	PushMessageStatusGameSecretChatRoomCreated     = "secret_chat_room_created"
	PushMessageStatusGameFinalSelectionStart       = "final_selection_start"
	PushMessageStatusGameFinalSelectionEnd         = "final_selection_end"
	PushMessageStatusGameFinalSelectionResult      = "final_selection_result"
	PushMessageStatusGameEnd                       = "end"
)

type Client struct {
	Conn *websocket.Conn
	Send chan interface{}
}

type Config struct {
	MatchClients sync.Map // key: userID, value: *websocket.Conn
	RedisClient  *redis.RedisClient
}

type MatchResponse struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id"`
}
