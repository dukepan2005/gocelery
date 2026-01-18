package gocelery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/google/uuid"
)

// CeleryMessageV2 is actual message to be sent to Redis
// https://docs.celeryproject.org/en/stable/internals/protocol.html
type CeleryMessageV2 struct {
	Body            string             `json:"body"`
	Headers         CeleryHeadersV2    `json:"headers"`
	Properties      CeleryPropertiesV2 `json:"properties"`
	ContentType     string             `json:"content-type"`
	ContentEncoding string             `json:"content-encoding"`
}

func (cm *CeleryMessageV2) reset() {
	cm.Headers.reset()
	cm.Body = ""
	cm.ContentType = "application/json"
	cm.ContentEncoding = "utf-8"
	cm.Properties.reset()
}

var celeryMessagePoolV2 = sync.Pool{
	New: func() interface{} {
		msg := &CeleryMessageV2{}
		msg.reset()
		return msg
	},
}

func getCeleryMessageV2(encodedTaskMessage string, headers CeleryHeadersV2) *CeleryMessageV2 {
	msg := celeryMessagePoolV2.Get().(*CeleryMessageV2)
	msg.Body = encodedTaskMessage
	msg.Headers = headers
	msg.Properties.CorrelationID = headers.ID

	return msg
}

func releaseCeleryMessageV2(v *CeleryMessageV2) {
	v.reset()
	celeryMessagePoolV2.Put(v)
}

// CeleryPropertiesV2 represents properties json
type CeleryPropertiesV2 struct {
	Priority      int                  `json:"priority"`
	BodyEncoding  string               `json:"body_encoding"`
	CorrelationID string               `json:"correlation_id"`
	ReplyTo       string               `json:"reply_to"`
	DeliveryInfo  CeleryDeliveryInfoV2 `json:"delivery_info"`
	DeliveryMode  int                  `json:"delivery_mode"`
	DeliveryTag   string               `json:"delivery_tag"`
}

func (cp *CeleryPropertiesV2) reset() {
	cp.Priority = 0
	cp.BodyEncoding = "base64"
	cp.CorrelationID = ""
	cp.ReplyTo = uuid.New().String()
	cp.DeliveryInfo = CeleryDeliveryInfoV2{
		RoutingKey: "celery",
		Exchange:   "",
	}
	cp.DeliveryMode = 2
	cp.DeliveryTag = uuid.New().String()
}

// CeleryDeliveryInfoV2 support celery v2 delivery info
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

// CeleryHeadersV2 support celery v2 header
type CeleryHeadersV2 struct {
	Lang     string `json:"lang"`
	Task     string `json:"task"`
	ID       string `json:"id"`
	RootID   string `json:"root_id"`
	ParentID string `json:"parent_id"`
	Group    string `json:"group"`

	Expires   interface{}    `json:"expires"`
	Shadow    interface{}    `json:"shadow"`
	Retries   int            `json:"retries"`
	Eta       interface{}    `json:"eta"`
	Argsrepr  string         `json:"argsrepr"`
	TimeLimit [2]interface{} `json:"timelimit"`

	Origin string `json:"origin"`
}

func (ch *CeleryHeadersV2) reset() {
	ch.Expires = nil
	ch.Shadow = nil
	ch.Lang = "py"
	ch.Retries = 0
	ch.Group = ""
	ch.ParentID = ""
	ch.Eta = nil
	ch.Argsrepr = ""
	ch.TimeLimit = [2]interface{}{60, nil}
	ch.RootID = ""
	ch.ID = ""
	ch.Task = ""
	ch.Origin = ""
}

var celeryHeadersPoolV2 = sync.Pool{
	New: func() interface{} {
		headers := &CeleryHeadersV2{}
		headers.reset()
		return headers
	},
}

func getCeleryMessageHeadersV2(task string, args ...interface{}) *CeleryHeadersV2 {
	return buildCeleryHeadersV2(task, args, nil)
}

func buildCeleryHeadersV2(task string, args []interface{}, kwargs map[string]interface{}) *CeleryHeadersV2 {
	headers := celeryHeadersPoolV2.Get().(*CeleryHeadersV2)
	hostname, _ := os.Hostname()
	headers.Origin = fmt.Sprintf("%d@%s", os.Getpid(), hostname)
	taskID := uuid.New().String()
	headers.ID, headers.RootID = taskID, taskID
	headers.Task = task
	headers.Argsrepr = formatArgsrepr(args, kwargs)
	return headers
}

func formatArgsrepr(args []interface{}, kwargs map[string]interface{}) string {
	if len(kwargs) == 0 {
		if args == nil {
			return "[]"
		}
		data, err := json.Marshal(args)
		if err != nil {
			return "[]"
		}
		return string(data)
	}
	payload := map[string]interface{}{
		"args":   args,
		"kwargs": kwargs,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "[]"
	}
	return string(data)
}

func releaseCeleryMessageHeadersV2(v *CeleryHeadersV2) {
	v.reset()
	celeryHeadersPoolV2.Put(v)
}

type embedStruct struct {
	Callbacks interface{} `json:"callbacks"`
	Errbacks  interface{} `json:"errbacks"`
	Chain     interface{} `json:"chain"`
	Chord     interface{} `json:"chord"`
}

// TaskMessageV2 is celery-compatible message protocol v2
// https://docs.celeryproject.org/en/stable/internals/protocol.html
type TaskMessageV2 struct {
	Args   []interface{}          `json:"args"`
	Kwargs map[string]interface{} `json:"kwargs"`
	Embed  embedStruct            `json:"embed"`
}

func (tm *TaskMessageV2) reset() {
	if tm.Args != nil {
		tm.Args = tm.Args[:0]
	}
	if tm.Kwargs == nil {
		tm.Kwargs = make(map[string]interface{})
	} else {
		for k := range tm.Kwargs {
			delete(tm.Kwargs, k)
		}
	}
	tm.Embed = embedStruct{}
}

var taskMessagePoolV2 = sync.Pool{
	New: func() interface{} {
		msg := &TaskMessageV2{}
		msg.reset()
		return msg
	},
}

func getTaskMessageV2(args ...interface{}) *TaskMessageV2 {
	return getTaskMessageV2WithKwargs(args, nil)
}

func getTaskMessageV2WithKwargs(args []interface{}, kwargs map[string]interface{}) *TaskMessageV2 {
	msg := taskMessagePoolV2.Get().(*TaskMessageV2)
	msg.Args = append(msg.Args[:0], args...)
	if kwargs == nil {
		for k := range msg.Kwargs {
			delete(msg.Kwargs, k)
		}
	} else {
		for k := range msg.Kwargs {
			delete(msg.Kwargs, k)
		}
		for k, v := range kwargs {
			msg.Kwargs[k] = v
		}
	}
	msg.Embed = embedStruct{}
	return msg
}

func releaseTaskMessageV2(v *TaskMessageV2) {
	v.reset()
	taskMessagePoolV2.Put(v)
}

// DecodeTaskMessageV2 decodes base64 encrypted body and return TaskMessage object
func DecodeTaskMessageV2(encodedBody string) (*TaskMessageV2, error) {
	body, err := base64.StdEncoding.DecodeString(encodedBody)
	if err != nil {
		return nil, err
	}
	message := taskMessagePoolV2.Get().(*TaskMessageV2)
	var payload []json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		releaseTaskMessageV2(message)
		return nil, err
	}
	if len(payload) != 3 {
		releaseTaskMessageV2(message)
		return nil, fmt.Errorf("unexpected task message payload length %d", len(payload))
	}

	var args []interface{}
	if err := json.Unmarshal(payload[0], &args); err != nil {
		releaseTaskMessageV2(message)
		return nil, err
	}
	message.Args = append(message.Args[:0], args...)

	var kwargs map[string]interface{}
	if err := json.Unmarshal(payload[1], &kwargs); err != nil {
		releaseTaskMessageV2(message)
		return nil, err
	}
	for k := range message.Kwargs {
		delete(message.Kwargs, k)
	}
	for k, v := range kwargs {
		message.Kwargs[k] = v
	}

	if err := json.Unmarshal(payload[2], &message.Embed); err != nil {
		releaseTaskMessageV2(message)
		return nil, err
	}

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
