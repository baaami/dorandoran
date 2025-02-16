package helper

import (
	"encoding/json"
	"log"
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
