package app

import (
	"errors"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
)

type recordedResponse struct {
	data     []byte
	empty    bool
	complete int
}

func (r *recordedResponse) Complete(data []byte) { r.data = data; r.complete++ }
func (r *recordedResponse) CompleteEmpty()       { r.empty = true; r.complete++ }

func send(e *engine, channel string, data []byte) *recordedResponse {
	r := &recordedResponse{}
	e.HandlePlatformMessage(&ui.PlatformMessage{Channel: channel, Data: data, Response: r})
	return r
}

func TestAssetChannelServesEngineAssets(t *testing.T) {
	prev := ui.EngineAssetLoader
	defer func() { ui.EngineAssetLoader = prev }()
	e := &engine{}
	ui.EngineAssetLoader = nil
	if r := send(e, "flutter/assets", []byte("a.png")); !r.empty || r.complete != 1 {
		t.Fatal("no loader", r)
	}
	var asked string
	ui.EngineAssetLoader = func(key string) ([]byte, error) {
		asked = key
		switch key {
		case "sub%20dir/a.png":
			return []byte("png"), nil
		case "empty":
			return []byte{}, nil
		}
		return nil, errors.New("missing")
	}
	if r := send(e, "flutter/assets", []byte("sub%20dir/a.png")); string(r.data) != "png" || r.empty || asked != "sub%20dir/a.png" {
		t.Fatal("asset", r, asked)
	}
	if r := send(e, "flutter/assets", []byte("missing")); !r.empty {
		t.Fatal("missing asset", r)
	}
	if r := send(e, "flutter/assets", []byte("empty")); !r.empty {
		t.Fatal("empty asset", r)
	}
}

func TestOtherChannelsGetEmptyReplies(t *testing.T) {
	e := &engine{}
	for _, channel := range []string{"flutter/textinput", "flutter/mousecursor", "flutter/platform"} {
		// Without a window the clipboard can't be reached: not implemented.
		if r := send(e, channel, []byte(`{"method":"Clipboard.hasStrings","args":"text/plain"}`)); !r.empty || r.complete != 1 {
			t.Fatal(channel, r)
		}
	}
}
