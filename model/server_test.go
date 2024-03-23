package model

import (
	"testing"

	"github.com/naiba/nezha/pkg/utils"
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
		if utils.Json.Unmarshal([]byte(serverStr), &serverRestore) != nil {
			t.Fatalf("Error: %s", serverStr)
		}
		if server.Name != serverRestore.Name {
			t.Fatalf("Expected %s, but got %s", server.Name, serverRestore.Name)
		}
	}
}
