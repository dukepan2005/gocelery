package gocelery

import (
	"fmt"
	"strings"
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

	if celeryTask.Embed.Callbacks != nil || celeryTask.Embed.Errbacks != nil || celeryTask.Embed.Chain != nil || celeryTask.Embed.Chord != nil {
		t.Error("the celeryTask.KwaEmbedrgs is not empty")
	}

}

func TestGetCeleryMessageHeaders(t *testing.T) {
	taskName := "account.tasks.visit_notify"
	headers := getCeleryMessageHeadersV2(taskName)

	if headers.ID == "" {
		t.Error("the headers ID in celery message protocol v1 can't be empty")
	}
	if headers.ID != headers.RootID {
		t.Errorf("the headers id <%s> is not equal headers root_id<%s>", headers.ID, headers.RootID)
	}

	if headers.Lang != "py" {
		t.Error("the headers lang in celery message protocol v1 != 'py'")
	}

	if headers.Task != taskName {
		t.Errorf("the headers task<%s> should be %s", headers.Task, taskName)
	}
}

func TestGetCeleryMessageV2(t *testing.T) {
	args := []interface{}{1965, 1970, "1", 1}
	celeryTask := getTaskMessageV2(args...)
	encodedTaskMessage, _ := celeryTask.Encode()

	taskName := "account.tasks.visit_notify"
	headers := getCeleryMessageHeadersV2(taskName, args...)
	celeryMessage := getCeleryMessageV2(encodedTaskMessage, *headers)

	fmt.Printf("%+v\n\n\n", headers)

	fmt.Printf("%+v\n\n\n", celeryMessage)
	if celeryMessage.Properties.CorrelationID != headers.ID {
		t.Errorf("the Properties.CorrelationID<%s> != headers['id']<%s>", celeryMessage.Properties.CorrelationID, headers.ID)
	}

	if celeryMessage.Body == "" {
		t.Error("message body should not be empty")
	}
	if celeryMessage.Body != encodedTaskMessage {
		t.Error("message body id not right")
	}
}

func TestGetTaskMessageV2WithKwargs(t *testing.T) {
	kwargs := map[string]interface{}{"user": "demo"}
	args := []interface{}{1, 2}
	celeryTask := getTaskMessageV2WithKwargs(args, kwargs)
	defer releaseTaskMessageV2(celeryTask)
	kwargs["user"] = "mutated"
	if celeryTask.Kwargs["user"] != "demo" {
		t.Fatal("kwarg mutations should not leak into pooled task")
	}
	if len(celeryTask.Args) != len(args) {
		t.Fatalf("expected %d args got %d", len(args), len(celeryTask.Args))
	}
}

func TestBuildCeleryHeadersV2WithKwargs(t *testing.T) {
	args := []interface{}{1965}
	kwargs := map[string]interface{}{"name": "demo"}
	headers := buildCeleryHeadersV2("account.tasks.visit_notify", args, kwargs)
	defer releaseCeleryMessageHeadersV2(headers)
	if !strings.Contains(headers.Argsrepr, "\"kwargs\"") {
		t.Error("argsrepr should encode kwargs payload")
	}
}
