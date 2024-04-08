package main

import (
	"slices"
	"testing"
)

type commandLineTestCase struct {
	cmdline string
	wantResult []string
}

func TestParseCommandLine(t *testing.T) {
	testCases := []commandLineTestCase{
		{
			cmdline: `setup "E:\Gamez\steamapps\common\Death's Door"`,
			wantResult: []string{"setup", "E:GamezsteamappscommonDeath's Door"},
		},
		{
			cmdline: `setup "E:\\Gamez\\steamapps\\common\\Death's Door"`,
			wantResult: []string{"setup", `E:\Gamez\steamapps\common\Death's Door`},
		},
	}
	for _, tt := range testCases {
		if got := parseCommandLine(tt.cmdline); !slices.Equal(got, tt.wantResult) {
			t.Errorf("cmdline %q:\n\tgot %q\n\twant %q", tt.cmdline, got, tt.wantResult)
		}
	}
}