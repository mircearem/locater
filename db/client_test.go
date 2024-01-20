package db

import (
	"encoding/json"
	"testing"
)

func TestPostIp(t *testing.T) {
	data := map[string]string{
		"5.3.199.181": "{\"lat\":45.99,\"lon\":23.25}",
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	client := NewClient(":7777")
	// Should maybe only post one at a time
	client.Post("iplatlona", bytes)
}
