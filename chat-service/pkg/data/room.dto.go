package data

import (
	"time"

	common "github.com/baaami/dorandoran/common/user"
)

type ChatRoomResponse struct {
	ID         string    `bson:"id" json:"id"` // UUID 사용
	Users      []common.User  `bson:"users" json:"users"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	ModifiedAt time.Time `bson:"modified_at" json:"modified_at"`
	// 추가적으로 각 사용자의 마지막 확인 메시지 ID를 추적하기 위한 필드를 고려할 수 있음
	UserLastRead map[string]time.Time `bson:"user_last_read" json:"user_last_read"`
}