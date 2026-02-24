package test

import (
	"os"
	"path/filepath"
	"testing"

	"comtradeviewer/comtrade"
)

func TestParseCFGMojibakeStationName(t *testing.T) {
	cfgPath := filepath.Join(".", "data", "mojibake.cfg")

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("failed to read mojibake cfg: %v", err)
	}

	meta, err := comtrade.ParseComtradeCFGFromBytes(cfgData)
	if err != nil {
		t.Fatalf("failed to parse mojibake cfg: %v", err)
	}
	t.Logf("Parsed station name: %q", meta.Station)

	if meta.Relay != "GDNZ209711079872" {
		t.Fatalf("unexpected relay: %q", meta.Relay)
	}
	if meta.Version != "1999" {
		t.Fatalf("unexpected version: %q", meta.Version)
	}
	if meta.AnalogChannelNum != 21 {
		t.Fatalf("unexpected analog channel num: %d", meta.AnalogChannelNum)
	}
	if meta.DigitalChannelNum != 41 {
		t.Fatalf("unexpected digital channel num: %d", meta.DigitalChannelNum)
	}
	if meta.Frequency != 50 {
		t.Fatalf("unexpected frequency: %v", meta.Frequency)
	}

	//  check the first analog channel name
	if len(meta.AnalogChannels) == 0 {
		t.Fatal("no analog channels found")
	}
	if meta.AnalogChannels[0].ChannelName != "保护电流A相" {
		t.Fatalf("unexpected first analog channel name: %q", meta.AnalogChannels[0].ChannelName)
	}
}
