package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerMarshal(t *testing.T) {
	patterns := []string{
		"asd > asd",
		"asd \" asd",
		"asd } asd",
	}

	for i := 0; i < len(patterns); i++ {
		server := Server{
			Name: patterns[i],
			Tag:  patterns[i],
		}
		serverStr := string(server.Marshal())
		var serverRestore Server
		assert.Nil(t, json.Unmarshal([]byte(serverStr), &serverRestore))
		assert.Equal(t, server, serverRestore)
	}
}
