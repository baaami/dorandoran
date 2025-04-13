package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/types/commontype"
	"solo/pkg/utils/stype"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
)

func ToJSON(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal data: %v", err)
		return nil
	}
	return json.RawMessage(bytes)
}

// 첫 번째 경로 요소를 추출하고 나머지 경로를 반환하는 함수
func ExtractFirstPath(path string) (string, string) {
	parts := strings.SplitN(path, "/", 3)

	if len(parts) > 1 {
		firstPath := parts[1]
		if len(parts) > 2 {
			return firstPath, "/" + parts[2]
		}
		return firstPath, "/"
	}

	return "", "/"
}

func IntToStringArray(arr []int) []string {
	return lo.Map(arr, func(item int, _ int) string {
		return strconv.Itoa(item)
	})
}

func StringToIntArrary(strSlice []string) ([]int, error) {
	var intSlice []int
	for _, str := range strSlice {
		num, err := strconv.Atoi(str)
		if err != nil {
			return nil, err // 변환 실패 시 에러 반환
		}
		intSlice = append(intSlice, num)
	}
	return intSlice, nil
}

// ["1:2", "3:4"] 형식으로 변환
func ConvertUserChoicesToMatchStrings(choices []stype.UserChoice) []string {
	matchStrings := make([]string, len(choices))

	for i, choice := range choices {
		// UserID:SelectedUserID 형식으로 문자열 생성
		matchStrings[i] = fmt.Sprintf("%d:%d",
			choice.UserID,
			choice.SelectedUserID,
		)
	}

	return matchStrings
}

// UserChoice 형식으로 변환
func ConvertMatchStringsToUserChoices(matchStrings []string) []stype.UserChoice {
	choices := make([]stype.UserChoice, len(matchStrings))
	for i, matchString := range matchStrings {
		parts := strings.Split(matchString, ":")
		if len(parts) != 2 {
			log.Printf("Invalid match string format: %s", matchString)
			continue
		}
		userID, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("Invalid user ID format: %s", parts[0])
			continue
		}
		selectedUserID, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("Invalid selected user ID format: %s", parts[1])
			continue
		}
		choices[i] = stype.UserChoice{
			UserID:         userID,
			SelectedUserID: selectedUserID,
		}
	}
	return choices
}

func GenerateMatchID(users []commontype.MatchedUser) string {
	timestamp := time.Now().Format("20060102150405")
	var userIDs []string
	for _, user := range users {
		userIDs = append(userIDs, strconv.Itoa(user.ID))
	}
	return fmt.Sprintf("%s_%s", timestamp, joinIDs(userIDs))
}

// 매칭된 유저 ID 목록 추출
func ExtractUserIDs(users []commontype.MatchedUser) []int {
	ids := make([]int, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}
	return ids
}

func joinIDs(ids []string) string {
	return strings.Join(ids, "_")
}
