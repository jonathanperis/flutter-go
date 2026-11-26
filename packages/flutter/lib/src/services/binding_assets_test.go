package services_test

// Ports of binding_test.dart's "Adds rootBundle LICENSES to
// LicenseRegistry" and "didHaveMemoryPressure clears asset caches"
// (Flutter 3.50.1), in this package's binding (its messenger is the
// TestDefaultBinaryMessenger, as TestBinding's is).

import (
	"bytes"
	"compress/gzip"
	"slices"
	"testing"

	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

const license1 = "L1Package1\nL1Package2\nL1Package3\n\nL1Paragraph1\n\nL1Paragraph2\n\nL1Paragraph3"
const license2 = "L2Package1\nL2Package2\nL2Package3\n\nL2Paragraph1\n\nL2Paragraph2\n\nL2Paragraph3"

var combinedLicenses = license1 + "\n--------------------------------------------------------------------------------\n" + license2 + "\n"

func TestBindingAddsRootBundleLicenses(t *testing.T) {
	ensureInitialized()
	messenger().SetMockMessageHandler("flutter/assets", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			if string(message) == "NOTICES.Z" {
				var buf bytes.Buffer
				w := gzip.NewWriter(&buf)
				w.Write([]byte(combinedLicenses))
				w.Close()
				return buf.Bytes()
			}
			return nil
		})
	})
	defer messenger().SetMockMessageHandler("flutter/assets", nil)
	runAsyncWaiting(t, func() {
		licenses := async.StreamToList(foundation.LicenseRegistry.Licenses()).Await()
		expect(len(licenses) == 2, "licenses %v", licenses)
		texts := func(e foundation.LicenseEntry) []string {
			var out []string
			for p := range e.Paragraphs() {
				out = append(out, p.Text)
			}
			return out
		}
		expect(slices.Equal(slices.Collect(licenses[0].Packages()), []string{"L1Package1", "L1Package2", "L1Package3"}), "packages 0")
		expect(slices.Equal(texts(licenses[0]), []string{"L1Paragraph1", "L1Paragraph2", "L1Paragraph3"}), "paragraphs 0 %q", texts(licenses[0]))
		expect(slices.Equal(slices.Collect(licenses[1].Packages()), []string{"L2Package1", "L2Package2", "L2Package3"}), "packages 1")
		expect(slices.Equal(texts(licenses[1]), []string{"L2Paragraph1", "L2Paragraph2", "L2Paragraph3"}), "paragraphs 1 %q", texts(licenses[1]))
		// The text after the packages' paragraph, as _parseLicenses splits it.
		raw, ok := licenses[0].(*foundation.LicenseEntryWithLineBreaks)
		expect(ok && raw.Text == "L1Paragraph1\n\nL1Paragraph2\n\nL1Paragraph3", "text %q", raw.Text)
	})
}

func TestBindingMemoryPressureClearsAssetCaches(t *testing.T) {
	ensureInitialized()
	flutterAssetsCallCount := 0
	messenger().SetMockMessageHandler("flutter/assets", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			flutterAssetsCallCount++
			return []byte("test_asset_data")
		})
	})
	defer messenger().SetMockMessageHandler("flutter/assets", nil)
	runAsync(t, func() {
		load := func(key string, want int) {
			s := services.RootBundle.LoadString(key).Await()
			expect(s == "test_asset_data" && flutterAssetsCallCount == want, "%s: %q, %d calls", key, s, flutterAssetsCallCount)
		}
		load("test_asset", 1)
		load("test_asset2", 2)
		load("test_asset", 2)
		load("test_asset2", 2)
		message := services.JSONMessageCodec{}.EncodeMessage(map[string]any{"type": "memoryPressure"})
		messenger().HandlePlatformMessage("flutter/system", message, func([]byte) {}).Await()
		load("test_asset", 3)
		load("test_asset2", 4)
		load("test_asset", 4)
		load("test_asset2", 4)
	})
}

func TestBindingEvictRemovesCachedAsset(t *testing.T) {
	ensureInitialized()
	calls := 0
	messenger().SetMockMessageHandler("flutter/assets", func(message []byte) async.Future[[]byte] {
		return async.Async(func() []byte {
			calls++
			return []byte("data")
		})
	})
	defer messenger().SetMockMessageHandler("flutter/assets", nil)
	runAsync(t, func() {
		services.RootBundle.LoadString("evicted").Await()
		services.RootBundle.LoadString("evicted").Await()
		expect(calls == 1, "cached: %d calls", calls)
		ensureInitialized().Evict("evicted") // ext.flutter.evict
		services.RootBundle.LoadString("evicted").Await()
		expect(calls == 2, "evicted: %d calls", calls)
	})
}

func TestBindingLifecycleNotifiesSystemChrome(t *testing.T) {
	ensureInitialized()
	_, done := logMethodCalls(services.SystemChannels.Platform)
	defer done()
	runAsync(t, func() {
		services.SystemChrome.SetSystemUIOverlayStyle(services.SystemUiOverlayStyleDark)
		idle()
		expect(services.SystemChrome.LatestStyle() != nil, "no latest style")
		// The engine reports the app detached: SystemChrome forgets the
		// style it sent.
		for _, state := range []string{"AppLifecycleState.resumed", "AppLifecycleState.detached"} {
			messenger().HandlePlatformMessage("flutter/lifecycle", services.StringCodec{}.EncodeMessage(&state), func([]byte) {}).Await()
		}
		idle()
		expect(services.SystemChrome.LatestStyle() == nil, "latest style kept: %v", services.SystemChrome.LatestStyle())
		resumed := "AppLifecycleState.resumed"
		messenger().HandlePlatformMessage("flutter/lifecycle", services.StringCodec{}.EncodeMessage(&resumed), func([]byte) {}).Await()
		idle()
	})
}
