package main

import (
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/google/go-cmp/cmp"
)

func TestCLITTL(t *testing.T) {
	type want struct {
		ttl    time.Duration
		hasErr bool
	}

	cases := map[string]struct {
		reason string
		args   []string
		want   want
	}{
		"Default": {
			reason: "The TTL should default to one minute",
			args:   []string{},
			want:   want{ttl: time.Minute},
		},
		"Custom": {
			reason: "The TTL should be parsed from the flag",
			args:   []string{"--ttl=5m"},
			want:   want{ttl: 5 * time.Minute},
		},
		"Zero": {
			reason: "A TTL of zero should be accepted, it disables caching",
			args:   []string{"--ttl=0s"},
			want:   want{ttl: 0},
		},
		"Negative": {
			reason: "A negative TTL should be rejected",
			args:   []string{"--ttl=-5m"},
			want:   want{hasErr: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cli := &CLI{}

			parser, err := kong.New(cli)
			if err != nil {
				t.Fatalf("kong.New(...): %v", err)
			}

			_, err = parser.Parse(tc.args)
			if (err != nil) != tc.want.hasErr {
				t.Fatalf("%s\nparser.Parse(%v): unexpected error state: %v", tc.reason, tc.args, err)
			}

			if tc.want.hasErr {
				return
			}

			if diff := cmp.Diff(tc.want.ttl, cli.TTL); diff != "" {
				t.Errorf("%s\nparser.Parse(%v): -want ttl, +got ttl:\n%s", tc.reason, tc.args, diff)
			}
		})
	}
}
