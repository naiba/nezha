package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testSt struct {
	input  string
	output string
}

func TestNotification(t *testing.T) {
	cases := []testSt{
		{
			input:  "ip(v4:103.80.236.249,v6:[d5ce:d811:cdb8:067a:a873:2076:9521:9d2d])",
			output: "ip(v4:103.****.249,v6:[d5ce:d811:****:9521:9d2d])",
		},
		{
			input:  "ip(v4:3.80.236.29,v6:[d5ce::cdb8:067a:a873:2076:9521:9d2d])",
			output: "ip(v4:3.****.29,v6:[d5ce::****:9521:9d2d])",
		},
		{
			input:  "ip(v4:3.80.236.29,v6:[d5ce::cdb8:067a:a873:2076::9d2d])",
			output: "ip(v4:3.****.29,v6:[d5ce::****::9d2d])",
		},
		{
			input:  "ip(v4:3.80.236.9,v6:[d5ce::cdb8:067a:a873:2076::9d2d])",
			output: "ip(v4:3.****.9,v6:[d5ce::****::9d2d])",
		},
		{
			input:  "ip(v4:3.80.236.9,v6:[d5ce::cdb8:067a:a873:2076::9d2d])",
			output: "ip(v4:3.****.9,v6:[d5ce::****::9d2d])",
		},
	}

	for _, c := range cases {
		assert.Equal(t, IPDesensitize(c.input), c.output)
	}
}
