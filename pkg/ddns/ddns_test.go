package ddns

import "testing"

type testSt struct {
	domain string
	zone   string
	prefix string
}

func TestSplitDomainSOA(t *testing.T) {
	cases := []testSt{
		{
			domain: "www.example.co.uk",
			zone:   "example.co.uk.",
			prefix: "www",
		},
		{
			domain: "abc.example.com",
			zone:   "example.com.",
			prefix: "abc",
		},
	}

	for _, c := range cases {
		prefix, zone, err := splitDomainSOA(c.domain)
		if err != nil {
			t.Fatalf("Error: %s", err)
		}
		if prefix != c.prefix {
			t.Fatalf("Expected prefix %s, but got %s", c.prefix, prefix)
		}
		if zone != c.zone {
			t.Fatalf("Expected zone %s, but got %s", c.zone, zone)
		}
	}
}
