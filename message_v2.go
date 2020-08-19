package gocelery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	uuid "github.com/satori/go.uuid"
)

// CeleryMessageV2 is actual message to be sent to Redis
// https://docs.celeryproject.org/en/stable/internals/protocol.html
type CeleryMessageV2 struct {
	Body            string                 `json:"body"`
	Headers         map[string]interface{} `json:"headers"`
	Properties      CeleryPropertiesV2     `json:"properties"`
	ContentType     string                 `json:"content-type"`
	ContentEncoding string                 `json:"content-encoding"`
}

func (cm *CeleryMessageV2) reset() {
	cm.Headers = nil
	cm.Body = ""
	cm.ContentEncoding = ""
	cm.Properties.CorrelationID = uuid.Must(uuid.NewV4()).String()
	cm.Properties.ReplyTo = uuid.Must(uuid.NewV4()).String()
	cm.Properties.DeliveryTag = uuid.Must(uuid.NewV4()).String()
}

var celeryMessagePoolV2 = sync.Pool{
	New: func() interface{} {
		return &CeleryMessageV2{
			Body: "",
			Headers: map[string]interface{}{
				"expires":   nil,
				"shadow":    nil,
				"lang":      "py",
				"retries":   0,
				"group":     nil,
				"parent_id": nil,
				"eta":       nil,
			},
			ContentType: "application/json",
			Properties: CeleryPropertiesV2{
				Priority:     0,
				BodyEncoding: "base64",
				// CorrelationID: uuid.Must(uuid.NewV4()).String(),
				ReplyTo: uuid.Must(uuid.NewV4()).String(),
				DeliveryInfo: CeleryDeliveryInfoV2{
					RoutingKey: "celery",
					Exchange:   "",
				},
				DeliveryMode: 2,
				DeliveryTag:  uuid.Must(uuid.NewV4()).String(),
			},
			ContentEncoding: "utf-8",
		}
	},
}

func getCeleryMessageV2(encodedTaskMessage string, headers map[string]interface{}) *CeleryMessageV2 {
	msg := celeryMessagePoolV2.Get().(*CeleryMessageV2)
	msg.Body = encodedTaskMessage
	msg.Headers = headers
	msg.Properties.CorrelationID = headers["id"].(string)

	return msg
}

func releaseCeleryMessageV2(v *CeleryMessageV2) {
	v.reset()
	celeryMessagePoolV2.Put(v)
}

// CeleryProperties represents properties json
type CeleryPropertiesV2 struct {
	Priority      int                  `json:"priority"`
	BodyEncoding  string               `json:"body_encoding"`
	CorrelationID string               `json:"correlation_id"`
	ReplyTo       string               `json:"reply_to"`
	DeliveryInfo  CeleryDeliveryInfoV2 `json:"delivery_info"`
	DeliveryMode  int                  `json:"delivery_mode"`
	DeliveryTag   string               `json:"delivery_tag"`
}

type CeleryDeliveryInfoV2 struct {
	RoutingKey string `json:"routing_key"`
	Exchange   string `json:"exchange"`
}

// GetTaskMessageV2 retrieve and decode task messages from broker
func (cm *CeleryMessageV2) GetTaskMessageV2() *TaskMessageV2 {
	// ensure content-type is 'application/json'
	if cm.ContentType != "application/json" {
		log.Println("unsupported content type " + cm.ContentType)
		return nil
	}
	// ensure body encoding is base64
	if cm.Properties.BodyEncoding != "base64" {
		log.Println("unsupported body encoding " + cm.Properties.BodyEncoding)
		return nil
	}
	// ensure content encoding is utf-8
	if cm.ContentEncoding != "utf-8" {
		log.Println("unsupported encoding " + cm.ContentEncoding)
		return nil
	}
	// decode body
	taskMessage, err := DecodeTaskMessageV2(cm.Body)
	if err != nil {
		log.Println("failed to decode task message")
		return nil
	}
	return taskMessage
}

type embedStruct struct {
	Callbacks interface{} `json:"callbacks"`
	Errbacks  interface{} `json:"errbacks"`
	Chain     interface{} `json:"chain"`
	Chord     interface{} `json:"chord"`
}

func getCeleryMessageHeaders(task string) map[string]interface{} {
	hostname, _ := os.Hostname()
	headers := map[string]interface{}{
		"expires": nil,
		"shadow":  nil,
		// "kwargsrepr": "{}",
		"lang":      "py",
		"group":     nil,
		"parent_id": nil,
		"eta":       nil,

		"retries":   0,
		"timelimit": [2]interface{}{60, nil},

		"root_id": "",
		// "argsrepr": "",
		"task": task,

		"origin": fmt.Sprintf("%d@%s", os.Getpid(), hostname),
		"id":     uuid.Must(uuid.NewV4()).String(),
	}
	// headers["argsrepr"] = args
	headers["root_id"] = headers["id"]

	return headers
}

// TaskMessageV2 is celery-compatible message protocol v2
// https://docs.celeryproject.org/en/stable/internals/protocol.html
type TaskMessageV2 struct {
	Args   []interface{}          `json:"args"`
	Kwargs map[string]interface{} `json:"kwargs"`
	Embed  embedStruct            `json:"embed"`
}

func (tm *TaskMessageV2) reset() {
	tm.Args = nil
	tm.Kwargs = nil
	tm.Embed = embedStruct{}
}

var taskMessagePoolV2 = sync.Pool{
	New: func() interface{} {
		return &TaskMessageV2{
			Args:   make([]interface{}, 0),
			Kwargs: make(map[string]interface{}),
			Embed: embedStruct{
				Chord:     nil,
				Callbacks: nil,
				Errbacks:  nil,
				Chain:     nil,
			},
		}
	},
}

func getTaskMessageV2(args ...interface{}) *TaskMessageV2 {
	msg := taskMessagePoolV2.Get().(*TaskMessageV2)
	msg.Args = args
	return msg
}

func releaseTaskMessageV2(v *TaskMessageV2) {
	v.reset()
	taskMessagePoolV2.Put(v)
}

// DecodeTaskMessage decodes base64 encrypted body and return TaskMessage object
func DecodeTaskMessageV2(encodedBody string) (*TaskMessageV2, error) {
	body, err := base64.StdEncoding.DecodeString(encodedBody)
	if err != nil {
		return nil, err
	}
	message := taskMessagePoolV2.Get().(*TaskMessageV2)
	messageArr := [3]interface{}{message.Args, message.Kwargs, message.Embed}

	err = json.Unmarshal(body, &messageArr)
	if err != nil {
		return nil, err
	}

	args := messageArr[0]
	kwargs := messageArr[1]
	embed := messageArr[2]
	message.Args = args.([]interface{})
	message.Kwargs = kwargs.(map[string]interface{})
	message.Embed = embed.(embedStruct)

	return message, nil
}

// Encode returns base64 json encoded string
func (tm *TaskMessageV2) Encode() (string, error) {
	if tm.Args == nil {
		tm.Args = make([]interface{}, 0)
	}
	messageArr := [3]interface{}{tm.Args, tm.Kwargs, tm.Embed}
	jsonData, err := json.Marshal(messageArr)
	if err != nil {
		return "", err
	}
	encodedData := base64.StdEncoding.EncodeToString(jsonData)
	return encodedData, err
}
