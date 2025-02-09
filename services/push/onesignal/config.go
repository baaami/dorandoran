package onesignal

type PushMessage struct {
	AppID          string            `json:"app_id"`
	IncludeAliases IncludeAliases    `json:"include_aliases"`
	TargetChannel  string            `json:"target_channel"`
	Headings       map[string]string `json:"headings"`
	Contents       map[string]string `json:"contents"`
	AppUrl         string            `json:"app_url"`
}

type IncludeAliases struct {
	ExternalID []string `json:"external_id"`
}

type Payload struct {
	PushUserList []int  `json:"push_user_list"`
	Header       string `json:"header"`
	Content      string `json:"content"`
	Url          string `json:"url"`
}

type ChatData struct {
	Msg    string `json:"msg"`
	RoomID string `json:"room_id"`
}

type RoomTimeoutData struct {
	RoomID string `json:"room_id"`
}

const (
	KindChat = iota
	KindRoomTimeout
)
