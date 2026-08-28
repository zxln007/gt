package main

import (
	"encoding/base64"
	"testing"
)

func TestDecodeImportPayloadRawYAML(t *testing.T) {
	raw := "remote:\n  - tcp://example.com:7001\nid: client\nsecret: token"

	got, err := decodeImportPayload(raw)
	if err != nil {
		t.Fatalf("decodeImportPayload returned error: %v", err)
	}
	if got != raw {
		t.Fatalf("decodeImportPayload mismatch:\nwant %q\ngot  %q", raw, got)
	}
}

func TestDecodeImportPayloadGTURL(t *testing.T) {
	raw := "remote:\n  - tcp://example.com:7001\nid: client\nsecret: token\n"
	encoded := base64.RawURLEncoding.EncodeToString([]byte(raw))

	got, err := decodeImportPayload("gt://import?profile=" + encoded)
	if err != nil {
		t.Fatalf("decodeImportPayload returned error: %v", err)
	}
	if got != raw {
		t.Fatalf("decodeImportPayload mismatch:\nwant %q\ngot  %q", raw, got)
	}
}

func TestRemoteDialAddress(t *testing.T) {
	tests := map[string]string{
		"tcp://example.com:7001": "example.com:7001",
		"example.com:7001":       "example.com:7001",
	}

	for input, want := range tests {
		got, err := remoteDialAddress(input)
		if err != nil {
			t.Fatalf("remoteDialAddress(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("remoteDialAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRemoteDialAddressRejectsMissingPort(t *testing.T) {
	if _, err := remoteDialAddress("tcp://example.com"); err == nil {
		t.Fatal("remoteDialAddress accepted URL without port")
	}
}
