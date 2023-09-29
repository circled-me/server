package push

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"server/config"
)

const (
	NotificationTypeNewAssetsInAlbum = 1
)

var httpClient = http.Client{}

type Notification struct {
	UserToken string `json:"user_token" binding:"required"`
	Title     string `json:"title" binding:"required"`
	Body      string `json:"body" binding:"required"`
	Data      Data   `json:"data"`
}

type Data struct {
	Type   int         `json:"type"`
	Detail interface{} `json:"detail"`
}

func Send(notification *Notification) error {
	buf := bytes.Buffer{}
	json.NewEncoder(&buf).Encode(*notification)
	resp, err := httpClient.Post(config.PUSH_SERVER+"/send", "application/json", &buf)
	if err != nil {
		log.Printf("SendPushNotification, error: %v", err)
		return err
	}
	if resp.StatusCode != 200 {
		buf.Reset()
		io.Copy(&buf, resp.Body)
		log.Printf("SendPushNotification error, status: %d, %s", resp.StatusCode, buf.String())
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	return nil
}
