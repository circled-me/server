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
	NotificationTypeNewAssetsInAlbum = "album"
	NotificationTypeCall             = "call"
)

var httpClient = http.Client{}

type Notification struct {
	Type       string            `json:"type"`
	UserTokens []string          `json:"user_tokens" binding:"required"`
	Title      string            `json:"title"`
	Body       string            `json:"body"`
	Data       map[string]string `json:"data"`
}

func (notification *Notification) SendTo(UserTokens []string) error {
	notification.UserTokens = UserTokens
	return notification.Send()
}

func (notification *Notification) Send() error {
	buf := bytes.Buffer{}
	json.NewEncoder(&buf).Encode(*notification)
	resp, err := httpClient.Post(config.PUSH_SERVER+"/send", "application/json", &buf)
	if err != nil {
		log.Printf("SendPushNotification, error: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		buf.Reset()
		io.Copy(&buf, resp.Body)
		log.Printf("SendPushNotification error, status: %d, %s", resp.StatusCode, buf.String())
		return fmt.Errorf("status: %d", resp.StatusCode)
	}
	return nil
}
