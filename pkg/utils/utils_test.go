package utils

import (
	"testing"
)

type testSt struct {
	input  string
	output string
}

func TestNotification(t *testing.T) {
	cases := []testSt{
		{
			input:  "103.80.236.249/d5ce:d811:cdb8:067a:a873:2076:9521:9d2d",
			output: "103.****.249/d5ce:d811:****:9521:9d2d",
		},
		{
			input:  "3.80.236.29/d5ce::cdb8:067a:a873:2076:9521:9d2d",
			output: "3.****.29/d5ce::****:9521:9d2d",
		},
		{
			input:  "3.80.236.29/d5ce::cdb8:067a:a873:2076::9d2d",
			output: "3.****.29/d5ce::****::9d2d",
		},
		{
			input:  "3.80.236.9/d5ce::cdb8:067a:a873:2076::9d2d",
			output: "3.****.9/d5ce::****::9d2d",
		},
		{
			input:  "3.80.236.9/d5ce::cdb8:067a:a873:2076::9d2d",
			output: "3.****.9/d5ce::****::9d2d",
		},
	}

	for _, c := range cases {
		if c.output != IPDesensitize(c.input) {
			t.Fatalf("Expected %s, but got %s", c.output, IPDesensitize(c.input))
		}
	}
}

func TestGenerGenerateRandomString(t *testing.T) {
	generatedString := make(map[string]bool)
	for i := 0; i < 100; i++ {
		str, err := GenerateRandomString(32)
		if err != nil {
			t.Fatalf("Error: %s", err)
		}
		if len(str) != 32 {
			t.Fatalf("Expected 32, but got %d", len(str))
		}
		if generatedString[str] {
			t.Fatalf("Duplicated string: %s", str)
		}
		generatedString[str] = true
	}
}
