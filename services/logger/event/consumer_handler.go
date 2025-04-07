package event

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"solo/pkg/logger"
	"solo/services/logger/repo"
)

type EventHandler struct {
	logRepo *repo.LogRepository
}

func NewEventHandler(logRepo *repo.LogRepository) *EventHandler {
	return &EventHandler{
		logRepo: logRepo,
	}
}

// HandleLogEvent는 로그 이벤트를 처리합니다
func (e *EventHandler) HandleLogEvent(payload json.RawMessage) {
	var baseLog logger.BaseLog
	if err := json.Unmarshal(payload, &baseLog); err != nil {
		log.Printf("❌ Failed to unmarshal log event: %v", err)
		return
	}

	// MongoDB에 로그 저장
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := e.logRepo.InsertLog(ctx, baseLog)
	if err != nil {
		log.Printf("❌ Failed to insert log: %v", err)
		return
	}

	log.Printf("✅ Log saved: %s", baseLog.Message)
}
