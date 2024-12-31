package webrtc

import (
	"encoding/json"
	"log"
	"server/utils"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	roomAbandonedTimeout = 1 * time.Minute
	userAbandonedTimeout = 20 * time.Second
)

type Client struct {
	conn     *websocket.Conn
	ID       string
	lastSeen time.Time
}

type Room struct {
	ID      string
	mutex   sync.RWMutex
	clients []*Client
}

var (
	rooms    = make(map[string]*Room)
	lastSeen = make(map[string]time.Time)
	mutex    sync.Mutex
)

func init() {
	go backgroundCleanup()
}

func ValidateRoom(room string) bool {
	mutex.Lock()
	_, exists := rooms[room]
	mutex.Unlock()
	return exists
}

func backgroundCleanup() {
	for {
		// Check for abandoned clients every 20 seconds
		mutex.Lock()
		checkAbandonedRooms()
		tmpRooms := make(map[string]*Room, len(rooms))
		for id, room := range rooms {
			tmpRooms[id] = room
		}
		mutex.Unlock()
		for _, room := range tmpRooms {
			abandoned := []string{}
			room.CheckAbandonedClients(func(clientId string) {
				abandoned = append(abandoned, clientId)
			})
			for _, clientId := range abandoned {
				room.Broadcast(clientId, map[string]interface{}{
					"type": "left",
					"from": clientId,
				})
				log.Printf("Client %s abandoned room\n", clientId)
			}
		}
		time.Sleep(userAbandonedTimeout)
	}
}

// checkAbandonedRooms checks for abandoned rooms and removes them (assumes the main mutex is locked)
func checkAbandonedRooms() {
	for id, last := range lastSeen {
		if time.Since(last) > roomAbandonedTimeout {
			log.Printf("Room %s has been abandoned\n", id)
			delete(rooms, id)
			delete(lastSeen, id)
		}
	}
}

func GetRoom(id string) *Room {
	mutex.Lock()
	defer mutex.Unlock()
	if room, exists := rooms[id]; exists {
		lastSeen[id] = time.Now()
		return room
	}
	room := &Room{
		ID: id,
	}
	rooms[id] = room
	lastSeen[id] = time.Now()
	return room
}

func (r *Room) CheckAbandonedClients(callback func(string)) {
	r.mutex.Lock()
	log.Printf("Checking for abandoned clients in room %s\n", r.ID)
	// Iterate over clients in reverse order to remove abandoned clients as there might be more than one
	for i := len(r.clients) - 1; i >= 0; i-- {
		client := r.clients[i]
		log.Printf("Client %s, last seen %v ago\n", client.ID, time.Since(client.lastSeen))
		if time.Since(client.lastSeen) > userAbandonedTimeout {
			callback(client.ID)
			r.clients = append(r.clients[:i], r.clients[i+1:]...)
			break
		}
	}
	r.mutex.Unlock()
}

func (r *Room) SeenClient(client *Client) {
	// Update the last seen time for the room
	mutex.Lock()
	lastSeen[r.ID] = time.Now()
	mutex.Unlock()
	// Update the last seen time for the client
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	for _, c := range r.clients {
		if c == client {
			c.lastSeen = time.Now()
			break
		}
	}
}

func (r *Room) SetUpClient(conn *websocket.Conn, id string) (client *Client, isNew bool) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// Is the client already in the room?
	for _, c := range r.clients {
		if c.ID == id {
			// Update the connection and last seen time
			c.conn = conn
			c.lastSeen = time.Now()
			return c, false
		}
	}
	// Add new client to the room
	client = &Client{
		conn:     conn,
		ID:       utils.Rand8BytesToBase62(),
		lastSeen: time.Now(),
	}
	r.clients = append(r.clients, client)
	return client, true
}

func (r *Room) RemoveClient(client *Client) {
	r.mutex.Lock()
	for i, c := range r.clients {
		if c == client {
			r.clients = append(r.clients[:i], r.clients[i+1:]...)
			break
		}
	}
	r.mutex.Unlock()
}

func (r *Room) Broadcast(from string, data interface{}) {
	message, _ := json.Marshal(data)
	r.mutex.RLock()
	for _, client := range r.clients {
		if client.ID != from {
			log.Printf("Sending (brodacast) message to %s: %s\n", client.ID, string(message))
			client.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
	r.mutex.RUnlock()
}

func (r *Room) MessageTo(clientId string, data interface{}) {
	message, _ := json.Marshal(data)
	r.mutex.RLock()
	for _, client := range r.clients {
		if client.ID == clientId {
			log.Printf("Sending (private) message to %s: %s\n", client.ID, string(message))
			client.conn.WriteMessage(websocket.TextMessage, message)
			break
		}
	}
	r.mutex.RUnlock()
}
