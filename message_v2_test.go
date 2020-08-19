package gocelery

import (
	"testing"
)

func TestGetTaskMessageV2(t *testing.T) {
	args := []interface{}{1965, 1970, 1, 1}
	celeryTask := getTaskMessageV2(args...)

	if len(celeryTask.Args) != len(args) {
		t.Errorf("the length of celeryTask.Args<%d> is not equal the length of args<%d>",
			len(celeryTask.Args), len(args))
	}

	if len(celeryTask.Kwargs) != 0 {
		t.Error("the length of celeryTask.Kwargs is not equal 0")
	}

	if celeryTask.Embed.callbacks != nil || celeryTask.Embed.errbacks != nil || celeryTask.Embed.
		chain != nil || celeryTask.Embed.chord != nil {
		t.Error("the celeryTask.KwaEmbedrgs is not empty")
	}

}

func TestGetCeleryMessageHeaders(t *testing.T) {
	taskName := "account.tasks.visit_notify"
	headers := getCeleryMessageHeaders(taskName)

	if headers["id"] == "" {
		t.Error("the headers ID in celery message protocol v1 can't be empty")
	}
	if headers["id"] != headers["root_id"] {
		t.Errorf("the headers id <%s> is not equal headers root_id<%s>", headers["id"], headers["root_id"])
	}

	if headers["lang"] != "py" {
		t.Error("the headers lang in celery message protocol v1 != 'py'")
	}

	if headers["task"] != taskName {
		t.Errorf("the headers task<%s> should be %s", headers["task"], taskName)
	}
}

func TestGetCeleryMessageV2(t *testing.T) {
	args := []interface{}{1965, 1970, 1, 1}
	celeryTask := getTaskMessageV2(args...)
	encodedTaskMessage, _ := celeryTask.Encode()

	taskName := "account.tasks.visit_notify"
	headers := getCeleryMessageHeaders(taskName)
	celeryMessage := getCeleryMessageV2(encodedTaskMessage, headers)

	if celeryMessage.Properties.CorrelationID != headers["id"] {
		t.Errorf("the Properties.CorrelationID<%s> != headers['id']<%s>", celeryMessage.Properties.CorrelationID, headers["id"])
	}

	if celeryMessage.Body == "" {
		t.Error("message body should not be empty")
	}
	if celeryMessage.Body != encodedTaskMessage {
		t.Error("message body id not right")
	}
}
