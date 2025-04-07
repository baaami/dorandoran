package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
)

type LogRepository struct {
	collection *mongo.Collection
}

func NewLogRepository(mongoClient *mongo.Client) (*LogRepository, error) {
	collection := mongoClient.Database("doran").Collection("logs")
	return &LogRepository{
		collection: collection,
	}, nil
}

// InsertLog는 로그를 MongoDB에 저장합니다
func (r *LogRepository) InsertLog(ctx context.Context, log interface{}) error {
	_, err := r.collection.InsertOne(ctx, log)
	return err
}

// InsertLogs는 여러 로그를 MongoDB에 저장합니다
func (r *LogRepository) InsertLogs(ctx context.Context, logs []interface{}) error {
	if len(logs) == 0 {
		return nil
	}

	_, err := r.collection.InsertMany(ctx, logs)
	return err
}
