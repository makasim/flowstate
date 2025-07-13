package netdriver

import (
	"testing"
)

func TestMarshalUnmarshalError(t *testing.T) {
	b := marshalError("", "")
	actCode, actMessage, err := unmarshalError(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actCode != "" {
		t.Errorf("expected code to be empty, got: %s", actCode)
	}
	if actMessage != "" {
		t.Errorf("expected message to be empty, got: %s", actMessage)
	}

	b = marshalError("theCode", "theMessage")
	actCode, actMessage, err = unmarshalError(b)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if actCode != "theCode" {
		t.Errorf("expected code to be 'theCode', got: %s", actCode)
	}
	if actMessage != "theMessage" {
		t.Errorf("expected message to be 'theMessage', got: %s", actMessage)
	}
}
