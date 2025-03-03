package main

import (
	"fmt"
	"log"

	"github.com/tbalthazar/onesignal-go"
)

func CreateNotifications(appID string, client *onesignal.Client) string {
	fmt.Println("### CreateNotifications ###")
	playerID := "c751f26e-22a0-4b87-9a21-96291678096b" // valid
	notificationReq := &onesignal.NotificationRequest{
		AppID:            appID,
		Contents:         map[string]string{"en": "Test message"},
		IncludePlayerIDs: []string{playerID},
	}

	createRes, res, err := client.Notifications.Create(notificationReq)
	if err != nil {
		fmt.Printf("--- res:%+v, err:%+v\n", res)
		log.Fatal(err)
	}
	fmt.Printf("--- createRes:%+v\n", createRes)
	fmt.Println()

	return createRes.ID
}

func main() {
	appID := "69a7a386-6f00-42e0-9791-d0dcbc7d3ccf"
	appKey := "os_v2_app_ngt2hbtpabbobf4r2doly7j4z72ntargfmqe4r4y6h6xymwbfkgrrd6kqf24x4432olcysbylex4xmi7ec7ujfohaaj5zrivu4p4biy"
	client := onesignal.NewClient(nil)
	client.AppKey = appKey

	notifID := CreateNotifications(appID, client)

	log.Printf("noti id: %s", notifID)
}
