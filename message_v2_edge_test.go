package gocelery

import (
	"testing"
)

// TestDecodeTaskMessageV2WithEmptyPayload tests decoding with invalid payload
func TestDecodeTaskMessageV2WithEmptyPayload(t *testing.T) {
	_, err := DecodeTaskMessageV2("")
	if err == nil {
		t.Error("expected error when decoding empty payload")
	}
}

// TestDecodeTaskMessageV2WithInvalidBase64 tests decoding with invalid base64
func TestDecodeTaskMessageV2WithInvalidBase64(t *testing.T) {
	_, err := DecodeTaskMessageV2("not-valid-base64!")
	if err == nil {
		t.Error("expected error when decoding invalid base64")
	}
}

// TestDecodeTaskMessageV2WithInvalidJSON tests decoding with invalid JSON
func TestDecodeTaskMessageV2WithInvalidJSON(t *testing.T) {
	// base64 encoded invalid JSON
	invalidJSON := "aW52YWxpZCBqc29u" // "invalid json"
	_, err := DecodeTaskMessageV2(invalidJSON)
	if err == nil {
		t.Error("expected error when decoding invalid JSON")
	}
}

// TestDecodeTaskMessageV2WithWrongPayloadLength tests wrong payload array length
func TestDecodeTaskMessageV2WithWrongPayloadLength(t *testing.T) {
	// Use base64 encoded payload with only 2 elements instead of 3
	// This is: [[1, 2], {"key": "value"}]
	invalidPayload := "W1sxLCAyXSwgeyJrZXkiOiAidmFsdWUifV0="

	_, err := DecodeTaskMessageV2(invalidPayload)
	if err == nil {
		t.Error("expected error when decoding payload with wrong length")
	}
}

// TestTaskMessageV2EncodeEmpty tests encoding with empty args
func TestTaskMessageV2EncodeEmpty(t *testing.T) {
	tm := getTaskMessageV2()
	defer releaseTaskMessageV2(tm)

	encoded, err := tm.Encode()
	if err != nil {
		t.Errorf("failed to encode empty task message: %v", err)
	}

	decoded, err := DecodeTaskMessageV2(encoded)
	if err != nil {
		t.Errorf("failed to decode empty task message: %v", err)
	}
	defer releaseTaskMessageV2(decoded)

	if len(decoded.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(decoded.Args))
	}
}

// TestGetTaskMessageV2WithKwargsNilArgs tests kwargs with nil args
func TestGetTaskMessageV2WithKwargsNilArgs(t *testing.T) {
	kwargs := map[string]interface{}{
		"timeout": 30,
		"retry":   true,
	}

	tm := getTaskMessageV2WithKwargs(nil, kwargs)
	defer releaseTaskMessageV2(tm)

	if len(tm.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(tm.Args))
	}

	if len(tm.Kwargs) != 2 {
		t.Errorf("expected 2 kwargs, got %d", len(tm.Kwargs))
	}
}

// TestFormatArgsreprEdgeCases tests formatArgsrepr with various inputs
func TestFormatArgsreprEdgeCases(t *testing.T) {
	// Test with nil args and nil kwargs
	result := formatArgsrepr(nil, nil)
	if result != "[]" {
		t.Errorf("expected '[]', got '%s'", result)
	}

	// Test with empty args and nil kwargs
	result = formatArgsrepr([]interface{}{}, nil)
	if result != "[]" {
		t.Errorf("expected '[]', got '%s'", result)
	}

	// Test with nil args and empty kwargs
	result = formatArgsrepr(nil, map[string]interface{}{})
	if result != "[]" {
		t.Errorf("expected '[]', got '%s'", result)
	}

	// Test with args and kwargs
	result = formatArgsrepr([]interface{}{1, 2}, map[string]interface{}{"key": "value"})
	if result == "[]" || result == "" {
		t.Errorf("expected non-empty result, got '%s'", result)
	}
}

// TestCeleryMessageV2Reset tests proper reset of pooled message
func TestCeleryMessageV2Reset(t *testing.T) {
	msg := getCeleryMessageV2("test-body", CeleryHeadersV2{
		Task: "test-task",
		ID:   "test-id",
	})

	if msg.Body != "test-body" {
		t.Errorf("expected body 'test-body', got '%s'", msg.Body)
	}

	releaseCeleryMessageV2(msg)

	// After release, the message should be reset
	if msg.Body != "" {
		t.Errorf("expected empty body after reset, got '%s'", msg.Body)
	}
}

// TestCeleryHeadersV2Reset tests proper reset of pooled headers
func TestCeleryHeadersV2Reset(t *testing.T) {
	headers := buildCeleryHeadersV2("test-task", []interface{}{1, 2}, nil)

	originalID := headers.ID
	if originalID == "" {
		t.Error("expected non-empty ID")
	}

	releaseCeleryMessageHeadersV2(headers)

	// After release, fields should be reset
	if headers.Task != "" {
		t.Errorf("expected empty task after reset, got '%s'", headers.Task)
	}
	if headers.ID != "" {
		t.Errorf("expected empty ID after reset, got '%s'", headers.ID)
	}
}
