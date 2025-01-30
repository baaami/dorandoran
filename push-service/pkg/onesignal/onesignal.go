package onesignal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/samber/lo"
)

func Push(payload Payload) error {
	appID := os.Getenv("ONESIGNAL_APP_ID")
	apiKey := os.Getenv("ONESIGNAL_API_KEY")

	if appID == "" || apiKey == "" {
		return fmt.Errorf("app id, app key is invalid, appid: %s, apikey: %s", appID, apiKey)
	}

	// OneSignal API URL
	url := "https://onesignal.com/api/v1/notifications"

	// samber/lo를 사용하여 userIDList를 string 배열로 변환
	externalIDs := lo.Map(payload.PushUserList, func(id int, _ int) string {
		return strconv.Itoa(id)
	})

	// PushMessage 구조체 초기화
	message := PushMessage{
		AppID: appID,
		IncludeAliases: IncludeAliases{
			ExternalID: externalIDs,
		},
		TargetChannel: "push",
		Headings: map[string]string{
			"en": payload.Header,
		},
		Contents: map[string]string{
			"en": payload.Content,
		},
		AppUrl: payload.Url,
	}

	// JSON 직렬화
	reqBody, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal reqBody: %w", err)
	}

	// HTTP 요청 생성
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// HTTP 헤더 설정
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", apiKey))

	// HTTP 요청 전송
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 응답 상태 확인
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	fmt.Printf("%s notification sent to users: %v", payload.Header, externalIDs)
	return nil
}
