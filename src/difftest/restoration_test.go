package difftest

// Go twin of flutter/test/restoration_test.dart: a RestorationManager on a
// mock engine and RestorationBucket operations; every 'put' is compared
// byte for byte (the maps' insertion order included).

import (
	"errors"
	"strconv"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	flutter_test "github.com/jonathanperis/flutter-go/packages/flutter_test/lib/src"
	"github.com/jonathanperis/flutter-go/src/async"
	"github.com/jonathanperis/flutter-go/src/dart"
)

func TestRestorationMatchesFlutter(t *testing.T) {
	runLoggedScenes(t, "flutter/build/restoration.json", func(add func(string)) {
		messenger := flutter_test.TestDefaultBinaryMessengerBindingInstance().DefaultBinaryMessenger().(*flutter_test.TestDefaultBinaryMessenger)
		type entry = dart.MapEntry[string, any]
		encode := func(data any) []byte { return services.NewStandardMessageCodec().EncodeMessage(data) }
		push := func(data any) {
			messenger.HandlePlatformMessage("flutter/restoration",
				services.NewStandardMethodCodec().EncodeMethodCall(services.NewMethodCall("push", data)), func([]byte) {}).Await()
		}
		flush := func(manager *services.RestorationManager, label string) {
			add("flush " + label)
			func() {
				defer func() {
					if r := recover(); r != nil {
						var fe *foundation.FlutterError
						if err, ok := r.(error); ok && errors.As(err, &fe) {
							add("FlutterError: " + fe.Message())
							return
						}
						panic(r)
					}
				}()
				manager.FlushData()
			}()
		}
		str := func(v any) string {
			if b, ok := v.(*services.RestorationBucket); ok && b == nil {
				return "null" // a typed nil pointer is Dart's null
			}
			return dart.ToString(v)
		}
		root := func(m *services.RestorationManager) *services.RestorationBucket { return m.RootBucket().Await() }

		var getReply any = []entry{
			{Key: "enabled", Value: true},
			{Key: "data", Value: encode([]entry{
				{Key: "v", Value: []entry{{Key: "count", Value: 7}, {Key: "name", Value: "Ada"}, {Key: "list", Value: []any{1, "two", 3.5, nil}}}},
				{Key: "c", Value: []entry{
					{Key: "page", Value: []entry{
						{Key: "v", Value: []entry{{Key: "scroll", Value: 120.5}, {Key: "flag", Value: true}}},
						{Key: "c", Value: []entry{
							{Key: "field", Value: []entry{
								{Key: "v", Value: []entry{{Key: "text", Value: "hello"}, {Key: "map", Value: []entry{{Key: "b", Value: 1}, {Key: "a", Value: 2}}}}},
							}},
						}},
					}},
					{Key: "other", Value: []entry{{Key: "v", Value: []entry{{Key: "x", Value: 1}}}}},
				}},
			})},
		}
		messenger.SetMockMethodCallHandler(services.SystemChannels.Restoration, func(call services.MethodCall) async.Future[any] {
			return async.Async(func() any {
				if call.Method == "put" {
					add("put " + joinBytes(call.Arguments.([]byte)))
					return nil
				}
				add("engine " + call.Method)
				return getReply
			})
		})

		manager := services.NewRestorationManager()
		notifications := 0
		manager.AddListener(func() { notifications++ })

		add("scene root")
		r := root(manager)
		add("root " + r.RestorationID() + " replacing " + strconv.FormatBool(r.IsReplacing()))
		for _, key := range []string{"count", "name", "list", "missing"} {
			add("read " + key + " " + str(r.Read(key)) + " contains " + strconv.FormatBool(r.Contains(key)))
		}

		add("scene values")
		r.Write("count", 7) // unchanged: nothing to serialize
		flush(manager, "unchanged")
		r.Write("count", 8)
		r.Write("added", "new")
		r.Write("nothing", nil)
		r.Write("pi", 3.14159)
		r.Write("nested", []any{dart.NewLinkedMap(dart.MapEntry[any, any]{Key: "z", Value: 1}, dart.MapEntry[any, any]{Key: "y", Value: []any{1, 2}}), false})
		flush(manager, "values")
		add("remove name " + str(r.Remove("name")))
		add("remove missing " + str(r.Remove("missing")))
		r.Write("count", 9)
		flush(manager, "remove")

		add("scene children")
		page := r.ClaimChild("page", "pageOwner")
		add("page scroll " + str(page.Read("scroll")) + " flag " + str(page.Read("flag")))
		field := page.ClaimChild("field", "fieldOwner")
		add("field text " + str(field.Read("text")) + " map " + str(field.Read("map")))
		flush(manager, "claimed existing")
		field.Write("text", "world")
		fresh := page.ClaimChild("fresh", "freshOwner")
		fresh.Write("a", 1)
		deep := fresh.ClaimChild("deep", "deepOwner")
		deep.Write("d", "deep")
		flush(manager, "new children")

		add("scene rename")
		fresh.Rename("renamed")
		flush(manager, "rename")
		fresh.Rename("renamed")
		flush(manager, "rename same")

		add("scene adopt")
		r.AdoptChild(deep)
		flush(manager, "adopt")
		empty := services.NewRestorationBucketEmpty("adopted", "emptyOwner")
		empty.Write("e", "mpty")
		page.AdoptChild(empty)
		flush(manager, "adopt empty")

		add("scene dispose")
		field.Dispose()
		flush(manager, "field disposed")
		page.Dispose()
		flush(manager, "page disposed")
		add("remove all " + str(r.Remove("count")) + " " + str(r.Remove("added")) + " " + str(r.Remove("nothing")) + " " +
			str(r.Remove("pi")) + " " + str(r.Remove("nested")) + " " + str(r.Remove("list")))
		flush(manager, "values removed")

		add("scene engine push")
		dupKeys := foundation.NewWriteBuffer()
		for _, b := range []int{13, 1, 7, 1, 118, 13, 3, 3} {
			dupKeys.PutUint8(b)
		}
		dupKeys.PutInt32(1)
		for _, b := range []int{7, 1, 97, 6} {
			dupKeys.PutUint8(b)
		}
		dupKeys.PutFloat64(1.0)
		for _, b := range []int{7, 1, 98, 7, 1, 107, 7, 1, 99} {
			dupKeys.PutUint8(b)
		}
		push([]entry{{Key: "enabled", Value: true}, {Key: "data", Value: dupKeys.Done()}})
		pushed := root(manager)
		add("notifications " + strconv.Itoa(notifications) + " same " + strconv.FormatBool(pushed == r) +
			" replacing " + strconv.FormatBool(manager.IsReplacing()))
		add("pushed 1 " + str(pushed.Read("1")) + " k " + str(pushed.Read("k")))
		pushed.Write("k", "C")
		pushed.Write("z", 26)
		flush(manager, "pushed")
		push([]entry{{Key: "enabled", Value: false}, {Key: "data", Value: nil}})
		add("notifications " + strconv.Itoa(notifications) + " root " + str(root(manager)))
		push([]entry{{Key: "enabled", Value: true}, {Key: "data", Value: nil}})
		blank := root(manager)
		add("notifications " + strconv.Itoa(notifications) + " blank " + strconv.FormatBool(blank.Contains("k")))
		blank.Write("only", 1)
		flush(manager, "blank")

		add("scene second manager")
		getReply = nil
		manager2 := services.NewRestorationManager()
		add("manager2 root " + str(root(manager2)))
		getReply = []entry{{Key: "enabled", Value: true}, {Key: "data", Value: nil}}
		manager3 := services.NewRestorationManager()
		root3 := root(manager3)
		a := root3.ClaimChild("a", nil)
		b := root3.ClaimChild("b", nil)
		b.Write("bv", 2)
		a.Write("av", 1)
		root3.Write("rv", 0)
		flush(manager3, "insertion order")
		b.Rename("a2")
		a.Rename("b2")
		flush(manager3, "renamed order")

		add("scene duplicate claims")
		manager4 := services.NewRestorationManager()
		root4 := root(manager4)
		other := root4.ClaimChild("other", "otherOwner")
		other.Write("x", 1)
		flush(manager4, "other")
		dup1 := root4.ClaimChild("other", "firstOther")
		add("dup1 x " + str(dup1.Read("x")))
		dup2 := root4.ClaimChild("other", "secondOther")
		add("dup2 x " + str(dup2.Read("x")))
		dup2.Write("y", 2)
		other.Dispose()
		flush(manager4, "two pending, one given up")
		func() {
			defer func() {
				if e, ok := recover().(*dart.AssertionError); ok {
					add("wedged: AssertionError(" + str(e.AssertionMessage()) + ")")
				}
			}()
			manager4.FlushData()
		}()
		manager2.Dispose()
		manager3.Dispose()
		manager.Dispose()
	})
}
