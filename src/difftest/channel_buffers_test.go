package difftest

// Go side of the ChannelBuffers differential test: the same scenes as
// flutter/test/channel_buffers_test.dart, on dart:ui's ChannelBuffers
// (ui.ChannelBuffers), compared line for line.

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func cbBD(s string) []byte { return append([]byte{}, s...) }

func cbShow(d []byte) string {
	if d == nil {
		return "null"
	}
	if len(d) == 0 {
		return "<empty>"
	}
	return string(d)
}

func cbReply(log func(string), name string) ui.PlatformMessageResponseCallback {
	return func(d []byte) { log("reply " + name + " " + cbShow(d)) }
}

func cbListener(log func(string), name string, respond bool) ui.ChannelCallback {
	return func(d []byte, cb ui.PlatformMessageResponseCallback) {
		log(name + " got " + cbShow(d))
		if respond {
			if d == nil {
				cb(nil)
			} else {
				cb(cbBD("re:" + cbShow(d)))
			}
		}
	}
}

func cbErr(r any) string {
	switch e := r.(type) {
	case dart.Exception:
		return e.Error()
	case dart.FormatException:
		return e.Error()
	case dart.StateError:
		return "StateError: " + e.Message
	case *dart.AssertionError:
		return "AssertionError null-name=" + strconv.FormatBool(strings.Contains(e.String(), "U+0000 NULL"))
	}
	return "error (other)"
}

func cbTry(log func(string), label string, f func()) {
	defer func() {
		if r := recover(); r != nil {
			log(label + ": " + cbErr(r))
		}
	}()
	f()
	log(label + " ok")
}

func cbStr(s string) []byte { return append([]byte{7, byte(len(s))}, s...) }

func cbCat(parts ...any) []byte {
	var out []byte
	for _, p := range parts {
		switch x := p.(type) {
		case []byte:
			out = append(out, x...)
		case int:
			out = append(out, byte(x))
		}
	}
	return out
}

// loggingBuffers is the Dart side's LoggingBuffers subclass.
type loggingBuffers struct {
	ui.ChannelBuffers
	log func(string)
}

func newLoggingBuffers(log func(string)) *loggingBuffers {
	b := &loggingBuffers{log: log}
	b.SetSelf(b)
	return b
}

func (b *loggingBuffers) Resize(name string, newSize int) {
	b.log("resize \"" + name + "\" " + strconv.Itoa(newSize))
	b.ChannelBuffers.Resize(name, newSize)
}

func (b *loggingBuffers) AllowOverflow(name string, allowed bool) {
	b.log("allowOverflow \"" + name + "\" " + strconv.FormatBool(allowed))
	b.ChannelBuffers.AllowOverflow(name, allowed)
}

var channelBuffersScenes = map[string]func(log func(string)){
	"push_default_capacity": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.Push("a", cbBD("1"), cbReply(log, "1"))
		b.Push("a", cbBD("2"), cbReply(log, "2"))
		b.Push("a", cbBD("3"), cbReply(log, "3"))
		b.Push("b", nil, cbReply(log, "b-null"))
		b.Push("b", cbBD(""), cbReply(log, "b-empty"))
		log("pushed")
		b.SetListener("a", cbListener(log, "A", true))
		b.SetListener("b", cbListener(log, "B", true))
		async.ScheduleMicrotask(func() { log("micro") })
		log("listeners set")
	},
	"resize_sequences": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.Resize("a", 4)
		for i := 1; i <= 6; i++ {
			b.Push("a", cbBD(strconv.Itoa(i)), cbReply(log, strconv.Itoa(i)))
		}
		log("resize 2")
		b.Resize("a", 2)
		log("resize 5")
		b.Resize("a", 5)
		b.Push("a", cbBD("7"), cbReply(log, "7"))
		log("resize 0")
		b.Resize("a", 0)
		b.Push("a", cbBD("8"), cbReply(log, "8"))
		log("resize 1")
		b.Resize("a", 1)
		b.Push("a", cbBD("9"), cbReply(log, "9"))
		b.Push("a", cbBD("10"), cbReply(log, "10"))
		cbTry(log, "resize -1", func() { b.Resize("a", -1) })
		b.Push("a", cbBD("11"), cbReply(log, "11"))
		b.Resize("a", 3)
		b.Push("a", cbBD("12"), cbReply(log, "12"))
		b.SetListener("a", cbListener(log, "A", false))
	},
	"drain_interleaved": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.Resize("a", 10)
		for i := 1; i <= 4; i++ {
			b.Push("a", cbBD(strconv.Itoa(i)), cbReply(log, strconv.Itoa(i)))
		}
		async.ScheduleMicrotask(func() { log("m1") })
		b.SetListener("a", func(d []byte, cb ui.PlatformMessageResponseCallback) {
			log("L1 got " + cbShow(d))
			if cbShow(d) == "1" {
				b.Push("a", cbBD("5"), cbReply(log, "5"))
				async.ScheduleMicrotask(func() { log("m-from-L1") })
			}
			if cbShow(d) == "2" {
				b.SetListener("a", func(d []byte, cb ui.PlatformMessageResponseCallback) {
					log("L2 got " + cbShow(d))
					cb(cbBD("L2"))
					if cbShow(d) == "4" {
						b.ClearListener("a")
						b.Push("a", cbBD("6"), cbReply(log, "6"))
						log("cleared")
					}
				})
			}
		})
		async.ScheduleMicrotask(func() { log("m2") })
		log("sync end")
		async.TimerRun(func() {
			log("timer: relisten")
			b.SetListener("a", cbListener(log, "L3", true))
			b.Push("a", cbBD("7"), cbReply(log, "7"))
			log("timer end")
		})
	},
	"listener_set_twice_before_drain": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.Resize("a", 3)
		b.Push("a", cbBD("1"), cbReply(log, "1"))
		b.Push("a", cbBD("2"), cbReply(log, "2"))
		b.SetListener("a", cbListener(log, "X", false))
		b.ClearListener("a")
		b.SetListener("a", cbListener(log, "Y", true))
		b.ClearListener("nope")
		async.ScheduleMicrotask(func() {
			log("m")
			b.ClearListener("a")
			b.Push("a", cbBD("3"), cbReply(log, "3"))
			b.Push("a", cbBD("4"), cbReply(log, "4"))
			b.SetListener("a", cbListener(log, "Z", true))
			b.Push("a", cbBD("5"), cbReply(log, "5"))
		})
	},
	"replace_listener_idle": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.SetListener("a", cbListener(log, "P", false))
		async.ScheduleMicrotask(func() {
			log("idle")
			b.SetListener("a", cbListener(log, "Q", true))
			b.Push("a", cbBD("1"), cbReply(log, "1"))
			b.Push("a", cbBD("2"), cbReply(log, "2"))
			log("pushed")
		})
	},
	"deprecated_drain": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.Resize("a", 5)
		for i := 1; i <= 3; i++ {
			b.Push("a", cbBD(strconv.Itoa(i)), cbReply(log, strconv.Itoa(i)))
		}
		async.OnComplete(b.Drain("a", func(d []byte, cb ui.PlatformMessageResponseCallback) async.Future[async.Void] {
			return async.AsyncVoid(func() {
				log("drain got " + cbShow(d))
				if cbShow(d) == "1" {
					b.Push("a", cbBD("4"), cbReply(log, "4"))
					async.FutureDelayed[async.Void](5*time.Millisecond, nil).Await()
					log("after delay")
				}
				cb(cbBD("d" + cbShow(d)))
			})
		}), func(async.Void) { log("drain done") }, nil)
		async.ScheduleMicrotask(func() { log("micro") })
		async.OnComplete(b.Drain("empty", func([]byte, ui.PlatformMessageResponseCallback) async.Future[async.Void] {
			return async.AsyncVoid(func() { log("never") })
		}), func(async.Void) { log("empty drain done") }, nil)
		log("sync end")
	},
	"allow_overflow": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.AllowOverflow("a", true)
		b.Push("a", cbBD("1"), cbReply(log, "1"))
		b.Push("a", cbBD("2"), cbReply(log, "2"))
		b.AllowOverflow("a", false)
		b.AllowOverflow("b", false)
		b.Push("b", cbBD("3"), cbReply(log, "3"))
		b.Resize("z", 0)
		b.AllowOverflow("z", true)
		b.Push("z", cbBD("4"), cbReply(log, "4"))
		b.SetListener("a", cbListener(log, "A", false))
		b.SetListener("b", cbListener(log, "B", false))
		b.SetListener("z", cbListener(log, "Z", false))
	},
	"handle_message": func(log func(string)) {
		b := newLoggingBuffers(log)
		cases := []struct {
			name string
			data []byte
		}{
			{"resize ok", cbCat(cbStr("resize"), 12, 2, cbStr("chan"), 3, 2, 0, 0, 0)},
			{"resize extra args", cbCat(cbStr("resize"), 12, 3, cbStr("chan"), 3, 1, 1, 0, 0, 1)},
			{"resize big", cbCat(cbStr("resize"), 12, 2, cbStr("x"), 3, 0xff, 0xff, 0xff, 0x7f)},
			{"resize unicode name", cbCat(cbStr("resize"), 12, 2, cbStr("ç✓"), 3, 7, 0, 0, 0)},
			{"overflow true", cbCat(cbStr("overflow"), 12, 2, cbStr("chan"), 1)},
			{"overflow false", cbCat(cbStr("overflow"), 12, 2, cbStr("chan"), 2)},
			{"method too long", []byte{7, 254, 0, 0}},
			{"unknown method", cbStr("flush")},
			{"resize not list", cbCat(cbStr("resize"), 13, 2)},
			{"resize one arg", cbCat(cbStr("resize"), 12, 1, cbStr("a"))},
			{"resize name not string", cbCat(cbStr("resize"), 12, 2, 3, 0, 0, 0, 0)},
			{"resize name too long", cbCat(cbStr("resize"), 12, 2, 7, 254, 0, 0)},
			{"resize null in name", cbCat(cbStr("resize"), 12, 2, cbStr("a\x00b"), 3, 1, 0, 0, 0)},
			{"resize size not int32", cbCat(cbStr("resize"), 12, 2, cbStr("a"), 4, 1, 0, 0, 0, 0, 0, 0, 0)},
			{"overflow not list", cbCat(cbStr("overflow"), 13)},
			{"overflow one arg", cbCat(cbStr("overflow"), 12, 1)},
			{"overflow name not string", cbCat(cbStr("overflow"), 12, 2, 0, 1)},
			{"overflow name too long", cbCat(cbStr("overflow"), 12, 2, 7, 255)},
			{"overflow flag not bool", cbCat(cbStr("overflow"), 12, 2, cbStr("a"), 0)},
			{"overflow flag int", cbCat(cbStr("overflow"), 12, 2, cbStr("a"), 3)},
			{"overflow null in name", cbCat(cbStr("overflow"), 12, 2, cbStr("a\x00"), 1)},
			{"truncated", []byte{7}},
			{"truncated resize", cbCat(cbStr("resize"), 12, 2, cbStr("a"), 3, 1)},
		}
		for _, c := range cases {
			cbTry(log, c.name, func() { b.HandleMessage(c.data) })
		}
		legacy := []string{
			"resize\ra\r3",
			"resize\r\r0",
			"resize\rb\r -0x1F ",
			"resize\rc\r+00042",
			"resize\rd\r9223372036854775807",
			"resize\ra",
			"resize\ra\r1\r2",
			"Resize\ra\r1",
			"resize\ra\rbar",
			"resize\ra\r",
			"resize\ra\r   ",
			"resize\ra\r-",
			"resize\ra\r+",
			"resize\ra\r0x",
			"resize\ra\r0xg",
			"resize\ra\r 0x1g",
			"resize\ra\r00012a",
			"resize\ra\r1.5",
			"resize\ra\r1234567890123456789x",
			"resize\ra\r12345678901234567890123456789012345678x",
			"resize\ra\r12345678901234567890",
			"resize\ra\r9223372036854775808",
			"resize\ra\r-9223372036854775808",
			"resize\ra\r-9223372036854775809",
			"resize\ra\r0xFFFFFFFFFFFFFFFF",
			"resize\ra\r0x10000000000000000",
			"resize\ra\r-0x8000000000000001",
			"resize\ra\r\u20007\u3000",
			"resize\ra\r١",
			"\ufeffresize\re\r2",
			"\ufeff\ufeffresize\re\r2",
			"resize\ra\r0x@",
			"resize\ra\r0x1`",
			"resize\ra\r0xaF",
			"resize\ra\r\u1680\u00a0\u0085 5\u202f\u205f\u2028\u2029\ufeff\u200a",
			"resize\ra\r\u2000\u200b5",
			"resize\ra\r5\u0084",
			"resize\rlong\r" + strings.Repeat("1", 70) + "x" + strings.Repeat("2", 30),
			"resize\rlong\r" + strings.Repeat("1", 100) + "x" + strings.Repeat("2", 100),
			"resize\rlong\r" + strings.Repeat("1", 10) + "x" + strings.Repeat("2", 100),
			"resize\rnul\x00l\r1",
			"",
		}
		for _, s := range legacy {
			cbTry(log, "legacy "+dart.JSONEncodeString(s), func() { b.HandleMessage(cbBD(s)) })
		}
		cbTry(log, "bad utf8", func() { b.HandleMessage([]byte{0xff, 0x61}) })
		cbTry(log, "utf8 ext", func() { b.HandleMessage([]byte{0x61, 0x80}) })
		cbTry(log, "utf8 unfinished", func() { b.HandleMessage([]byte{0x61, 0xe2, 0x82}) })
		cbTry(log, "utf8 missing ext", func() { b.HandleMessage([]byte{0x61, 0xe2, 0x41}) })
		cbTry(log, "utf8 overlong", func() { b.HandleMessage([]byte{0x61, 0xc0, 0x80}) })
		cbTry(log, "utf8 surrogate", func() { b.HandleMessage([]byte{0x61, 0xed, 0xa0, 0x80}) })
		cbTry(log, "utf8 range", func() { b.HandleMessage([]byte{0x61, 0xf4, 0x90, 0x80, 0x80}) })
		cbTry(log, "utf8 C1", func() { b.HandleMessage([]byte{0x61, 0xc1, 0xbf}) })
		cbTry(log, "utf8 F5", func() { b.HandleMessage([]byte{0x61, 0xf5, 0x80, 0x80, 0x80}) })
		cbTry(log, "utf8 overlong3", func() { b.HandleMessage([]byte{0x61, 0xe0, 0x80, 0x80}) })
		cbTry(log, "utf8 overlong4", func() { b.HandleMessage([]byte{0xf0, 0x80, 0x80, 0x80}) })
		cbTry(log, "utf8 bom only", func() { b.HandleMessage([]byte{0xef, 0xbb, 0xbf}) })
		cbTry(log, "utf8 partial bom", func() { b.HandleMessage([]byte{0xef, 0xbb}) })
		cbTry(log, "empty", func() { b.HandleMessage([]byte{}) })
		// The resized channels behave as resized.
		for i := 1; i <= 3; i++ {
			n := "c" + strconv.Itoa(i)
			b.Push("chan", cbBD(n), cbReply(log, n))
		}
		b.SetListener("chan", cbListener(log, "C", false))
	},
	"null_names": func(log func(string)) {
		b := ui.NewChannelBuffers()
		cbTry(log, "push", func() { b.Push("a\x00", cbBD("1"), cbReply(log, "1")) })
		cbTry(log, "setListener", func() { b.SetListener("a\x00", cbListener(log, "A", false)) })
		cbTry(log, "resize new", func() { b.Resize("a\x00", 2) })
		cbTry(log, "allowOverflow new", func() { b.AllowOverflow("a\x00", true) })
		cbTry(log, "allowOverflow false", func() { b.AllowOverflow("a\x00", false) })
		cbTry(log, "clearListener", func() { b.ClearListener("a\x00") })
	},
	"responses": func(log func(string)) {
		b := ui.NewChannelBuffers()
		b.SetListener("a", func(d []byte, cb ui.PlatformMessageResponseCallback) {
			log("A got " + cbShow(d))
			switch cbShow(d) {
			case "late":
				async.NewTimer(3*time.Millisecond, func() { cb(cbBD("late reply")) })
			case "twice":
				cb(cbBD("first"))
				cb(nil)
			case "never":
			default:
				cb(d)
			}
		})
		log("direct push:")
		b.Push("a", cbBD("now"), cbReply(log, "now"))
		b.Push("a", cbBD("late"), cbReply(log, "late"))
		b.Push("a", cbBD("twice"), cbReply(log, "twice"))
		b.Push("a", cbBD("never"), cbReply(log, "never"))
		b.Push("a", nil, cbReply(log, "null"))
		b.Push("a", cbBD(""), cbReply(log, "empty"))
		log("sync end")
	},
	"channel_buffers_singleton": func(log func(string)) {
		log("kDefaultBufferSize " + strconv.Itoa(ui.ChannelBuffersKDefaultBufferSize))
		log("kControlChannelName " + ui.ChannelBuffersKControlChannelName)
		log("identical " + strconv.FormatBool(ui.ChannelBuffersInstance() == ui.ChannelBuffersInstance()))
	},
}

func TestChannelBuffersMatchesDart(t *testing.T) {
	data, err := os.ReadFile("flutter/build/channel_buffers.json")
	if err != nil {
		t.Skipf("no reference (%v); run scripts/difftest.sh", err)
	}
	var ref map[string][]string
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	if len(ref) != len(channelBuffersScenes) {
		t.Fatalf("reference has %d scenes, Go has %d", len(ref), len(channelBuffersScenes))
	}
	for name, scene := range channelBuffersScenes {
		l := async.NewLoop()
		restore := async.SetCurrent(l)
		var got []string
		scene(func(s string) { got = append(got, s) })
		l.FlushMicrotasks()
		l.FlushTimers(time.Hour, true)
		restore()
		if !reflect.DeepEqual(got, ref[name]) {
			t.Errorf("%s:\n Go:   %q\n Dart: %q", name, got, ref[name])
		}
	}
}
