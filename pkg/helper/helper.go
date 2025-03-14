package helper

import (
	"encoding/json"
	"fmt"
	"log"
	"solo/pkg/utils/stype"
	"strconv"
	"strings"

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
