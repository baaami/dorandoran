package logger

import (
	"encoding/json"
	"os"
	"time"

	"solo/pkg/helper"
	"solo/pkg/mq"

	eventtypes "solo/pkg/types/eventtype"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Logger는 전역 로거 인스턴스
	Logger zerolog.Logger
	// RabbitMQ는 메시지 큐 인스턴스
	RabbitMQ *mq.RabbitMQ
	// currentService는 현재 서비스 타입을 저장합니다
	currentService ServiceType
)

const (
	ServiceTypeGateway ServiceType = iota
	ServiceTypeAuth
	ServiceTypeUser
	ServiceTypeChat
	ServiceTypeGame
	ServiceTypeMatch
	ServiceTypePush
)

// ServiceType은 서비스 타입을 나타내는 정수입니다
type ServiceType int

const (
	// 게임 관련 이벤트
	LogEventGameRoomCreate LogEventType = iota
	LogEventGameRoomTimeout
	LogEventGameStart
	LogEventGameEnd
	LogEventBalanceVote
	LogEventFinalChoice

	// 매칭 관련 이벤트
	LogEventMatchSuccess
	LogEventMatchFail

	// 커플 매칭 이벤트
	LogEventCoupleRoomCreate

	// 경고 이벤트
	LogEventWarning

	// 에러 이벤트
	LogEventError
)

// LogEventType은 로그 이벤트 타입을 나타내는 정수입니다
type LogEventType int

// BaseLog는 로그의 기본 구조를 정의합니다
type BaseLog struct {
	Level        string      `json:"level"`
	Timestamp    time.Time   `json:"timestamp"`
	Service      int         `json:"service"`
	LogEventType int         `json:"log_event_type"`
	Message      string      `json:"message"`
	Log          interface{} `json:"log"`
}

// InitLogger는 로거를 초기화합니다
func InitLogger(serviceType ServiceType) error {
	// RabbitMQ 연결
	var err error
	RabbitMQ, err = mq.ConnectToRabbitMQ()
	if err != nil {
		return err
	}

	// Exchange 선언
	err = RabbitMQ.DeclareExchange(mq.ExchangeLog, mq.ExchangeTypeFanout)
	if err != nil {
		return err
	}

	// 현재 서비스 타입 저장
	currentService = serviceType

	// 로그 포맷 설정
	zerolog.TimeFieldFormat = time.RFC3339
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}

	// 로거 설정
	Logger = zerolog.New(output).
		Level(zerolog.InfoLevel).
		With().
		Int("service", int(serviceType)).
		Timestamp().
		Logger()

	return nil
}

// Log는 BaseLog 구조체와 동일한 형식으로 로그를 출력합니다
func Log(level string, logEventType LogEventType, message string, logData interface{}) {
	// BaseLog 구조체 생성
	baseLog := BaseLog{
		Level:        level,
		Timestamp:    time.Now(),
		Service:      int(currentService),
		LogEventType: int(logEventType),
		Message:      message,
		Log:          logData,
	}

	eventPayload := eventtypes.EventPayload{
		EventType: eventtypes.EventTypeLog,
		Data:      helper.ToJSON(baseLog),
	}

	// JSON으로 변환
	jsonData, err := json.Marshal(eventPayload)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal log data")
		return
	}

	// RabbitMQ로 발행
	err = RabbitMQ.PublishMessage(mq.ExchangeLog, "", jsonData)
	if err != nil {
		log.Error().Err(err).Msg("Failed to publish log message")
		return
	}
}

// Debug는 debug 레벨 로그를 출력합니다
func Debug(logEventType LogEventType, message string, logData interface{}) {
	Log("debug", logEventType, message, logData)
}

// Info는 info 레벨 로그를 출력합니다
func Info(logEventType LogEventType, message string, logData interface{}) {
	Log("info", logEventType, message, logData)
}

// Warn은 warn 레벨 로그를 출력합니다
func Warn(logEventType LogEventType, message string, logData interface{}) {
	Log("warn", logEventType, message, logData)
}

// Error는 error 레벨 로그를 출력합니다
func Error(logEventType LogEventType, message string, logData interface{}) {
	Log("error", logEventType, message, logData)
}

// WithContext는 추가 컨텍스트를 포함한 로거를 반환합니다
func WithContext(fields map[string]interface{}) zerolog.Logger {
	return Logger.With().Fields(fields).Logger()
}
