package services_test

// Port of packages/flutter/test/services/system_context_menu_controller_test.dart
// (Flutter 3.50.1) with test/system_context_menu_utils.dart.
//
// Skipped: "showing a controller for an EditableText" (pumps an
// EditableText; widgets are not ported).

import (
	"reflect"
	"testing"

	ui "github.com/jonathanperis/flutter-go/engine/src/flutter/lib/ui/go"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/foundation"
	"github.com/jonathanperis/flutter-go/packages/flutter/lib/src/services"
	"github.com/jonathanperis/flutter-go/src/async"
)

// systemContextMenuItemDataFromJSON is system_context_menu_utils.dart's
// systemContextMenuItemDataFromJson.
func systemContextMenuItemDataFromJSON(json map[string]any) services.IOSSystemContextMenuItemData {
	title, _ := json["title"].(string)
	switch json["type"] {
	case "copy":
		return services.IOSSystemContextMenuItemDataCopy{}
	case "cut":
		return services.IOSSystemContextMenuItemDataCut{}
	case "paste":
		return services.IOSSystemContextMenuItemDataPaste{}
	case "selectAll":
		return services.IOSSystemContextMenuItemDataSelectAll{}
	case "searchWeb":
		return services.NewIOSSystemContextMenuItemDataSearchWeb(title)
	case "share":
		return services.NewIOSSystemContextMenuItemDataShare(title)
	case "lookUp":
		return services.NewIOSSystemContextMenuItemDataLookUp(title)
	case "captureTextFromCamera":
		return services.IOSSystemContextMenuItemDataLiveText{}
	case "custom":
		return services.NewIOSSystemContextMenuItemDataCustom(title, func() {})
	}
	panic(foundation.NewFlutterError("Invalid json for IOSSystemContextMenuItem.type"))
}

// contextMenuTest is the file's setUp (a mock clipboard holding data) and
// tearDown, with an active text input connection (required to show the
// system menu).
func contextMenuTest(t *testing.T, body func()) {
	ensureInitialized()
	mock := &mockClipboard{}
	messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, mock.handleMethodCall)
	defer messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, nil)
	runAsync(t, func() {
		services.Clipboard.SetData(services.NewClipboardData("Clipboard data")).Await()
		client := newFakeTextInputClient(tev("test1"))
		connection := services.TextInputAttach(client, client.configuration())
		defer connection.Close()
		body()
	})
}

// recordMenuCalls installs a platform mock recording the target rects
// shown, the items shown and the hides.
func recordMenuCalls() (rects *[]map[string]any, items *[][]services.IOSSystemContextMenuItemData, hideCount *int) {
	rects, items, hideCount = &[]map[string]any{}, &[][]services.IOSSystemContextMenuItemData{}, new(int)
	messenger().SetMockMethodCallHandler(services.SystemChannels.Platform, func(call services.MethodCall) async.Future[any] {
		return async.Async(func() any {
			switch call.Method {
			case "ContextMenu.showSystemContextMenu":
				arguments := call.Arguments.(map[string]any)
				*rects = append(*rects, arguments["targetRect"].(map[string]any))
				if raw, ok := arguments["items"].([]any); ok {
					var list []services.IOSSystemContextMenuItemData
					for _, v := range raw {
						list = append(list, systemContextMenuItemDataFromJSON(v.(map[string]any)))
					}
					*items = append(*items, list)
				}
			case "ContextMenu.hideSystemContextMenu":
				*hideCount++
			}
			return nil
		})
	})
	return
}

func rectIs(r map[string]any, rect ui.Rect) bool {
	return reflect.DeepEqual(r, map[string]any{"x": rect.Left, "y": rect.Top, "width": rect.Width(), "height": rect.Height()})
}

func mustPanicAssertion(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			panic("expected an AssertionError")
		}
	}()
	f()
}

var copyItems = []services.IOSSystemContextMenuItemData{services.IOSSystemContextMenuItemDataCopy{}}

func TestShowingAndHidingOneController(t *testing.T) {
	contextMenuTest(t, func() {
		rects, _, hideCount := recordMenuCalls()
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		expect(len(*rects) == 0 && *hideCount == 0 && !controller.IsVisible(), "initial")

		rect1 := ui.NewRectFromLTWH(0, 0, 100, 100)
		controller.ShowWithItems(rect1, copyItems)
		expect(len(*rects) == 1 && rectIs((*rects)[0], rect1), "%v", *rects)
		// Showing the same thing again does nothing.
		controller.ShowWithItems(rect1, copyItems)
		expect(controller.IsVisible() && len(*rects) == 1, "%v", *rects)
		// Showing a new rect calls the platform.
		rect2 := ui.NewRectFromLTWH(1, 1, 200, 200)
		controller.ShowWithItems(rect2, copyItems)
		expect(len(*rects) == 2 && rectIs((*rects)[1], rect2), "%v", *rects)
		// Hiding calls the platform.
		controller.Hide()
		expect(!controller.IsVisible() && *hideCount == 1, "hide %d", *hideCount)
		// Hiding again does nothing.
		controller.Hide()
		expect(!controller.IsVisible() && *hideCount == 1, "hide %d", *hideCount)
		// Showing the last shown rect calls the platform.
		controller.ShowWithItems(rect2, copyItems)
		expect(controller.IsVisible() && len(*rects) == 3 && rectIs((*rects)[2], rect2), "%v", *rects)
		controller.Hide()
		expect(!controller.IsVisible() && *hideCount == 2, "hide %d", *hideCount)
	})
}

func TestTheSystemCanHideTheMenuWithHandleSystemHide(t *testing.T) {
	contextMenuTest(t, func() {
		rects, _, hideCount := recordMenuCalls()
		systemHideCount := 0
		controller := services.NewSystemContextMenuController(services.SystemContextMenuControllerOptions{
			OnSystemHide: func() { systemHideCount++ },
		})
		defer controller.Dispose()
		expect(!controller.IsVisible() && len(*rects) == 0 && *hideCount == 0 && systemHideCount == 0, "initial")
		rect1 := ui.NewRectFromLTWH(0, 0, 100, 100)
		controller.ShowWithItems(rect1, copyItems).Await()
		expect(controller.IsVisible() && len(*rects) == 1 && rectIs((*rects)[0], rect1), "%v", *rects)

		// If the system hides the menu, onSystemHide is called.
		messenger().HandlePlatformMessage("flutter/platform",
			jsonMessage.EncodeMessage(map[string]any{"method": "ContextMenu.onDismissSystemContextMenu"}), func([]byte) {}).Await()
		expect(!controller.IsVisible() && *hideCount == 0 && systemHideCount == 1, "%v %d %d", controller.IsVisible(), *hideCount, systemHideCount)

		// Hiding does not call the platform, since the menu was already hidden.
		controller.Hide().Await()
		expect(!controller.IsVisible() && *hideCount == 0, "hide %d", *hideCount)
	})
}

func TestShowingASecondControllerWhileOneIsVisibleIsAnError(t *testing.T) {
	contextMenuTest(t, func() {
		controller1 := services.NewSystemContextMenuController()
		defer controller1.Dispose()
		expect(!controller1.IsVisible(), "visible")
		controller1.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), copyItems)
		expect(controller1.IsVisible(), "not visible")

		controller2 := services.NewSystemContextMenuController()
		defer controller2.Dispose()
		expect(controller1.IsVisible() && !controller2.IsVisible(), "visibility")
		mustPanicAssertion(func() { controller2.ShowWithItems(ui.NewRectFromLTWH(1, 1, 200, 200), copyItems) })
		expect(controller1.IsVisible() && !controller2.IsVisible(), "visibility")

		controller1.Hide()
		expect(!controller1.IsVisible() && !controller2.IsVisible(), "visibility")
	})
}

func TestShowingAndHidingTwoControllers(t *testing.T) {
	contextMenuTest(t, func() {
		rects, _, hideCount := recordMenuCalls()
		controller1 := services.NewSystemContextMenuController()
		defer controller1.Dispose()
		expect(!controller1.IsVisible() && len(*rects) == 0 && *hideCount == 0, "initial")
		rect1 := ui.NewRectFromLTWH(0, 0, 100, 100)
		controller1.ShowWithItems(rect1, copyItems)
		expect(controller1.IsVisible() && len(*rects) == 1 && (*rects)[0]["x"] == rect1.Left, "%v", *rects)
		controller1.Hide()
		expect(!controller1.IsVisible() && *hideCount == 1, "hide")

		controller2 := services.NewSystemContextMenuController()
		defer controller2.Dispose()
		expect(!controller2.IsVisible(), "visible")
		rect2 := ui.NewRectFromLTWH(1, 1, 200, 200)
		controller2.ShowWithItems(rect2, copyItems)
		expect(!controller1.IsVisible() && controller2.IsVisible() && len(*rects) == 2 && rectIs((*rects)[1], rect2), "%v", *rects)

		// Hiding the old controller does nothing.
		controller1.Hide()
		expect(!controller1.IsVisible() && controller2.IsVisible() && *hideCount == 1, "hide")
		// Hiding the new controller calls the platform.
		controller2.Hide()
		expect(!controller1.IsVisible() && !controller2.IsVisible() && *hideCount == 2, "hide")
	})
}

func itemListsEqual(a, b []services.IOSSystemContextMenuItemData) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !services.IOSSystemContextMenuItemDataEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestShowingAControllerWithCustomItems(t *testing.T) {
	contextMenuTest(t, func() {
		_, received, _ := recordMenuCalls()
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		expect(!controller.IsVisible(), "visible")
		rect := ui.NewRectFromLTWH(0, 0, 100, 100)
		items1 := []services.IOSSystemContextMenuItemData{
			services.IOSSystemContextMenuItemDataCut{}, services.IOSSystemContextMenuItemDataCopy{},
			services.IOSSystemContextMenuItemDataPaste{}, services.IOSSystemContextMenuItemDataSelectAll{},
			services.NewIOSSystemContextMenuItemDataSearchWeb("Special Search"),
		}
		controller.ShowWithItems(rect, items1)
		expect(controller.IsVisible() && len(*received) == 1 && itemListsEqual((*received)[0], items1), "%v", *received)
		// Showing the same thing again does nothing.
		controller.ShowWithItems(rect, items1)
		expect(controller.IsVisible() && len(*received) == 1, "%v", *received)
		// Showing new items calls the platform.
		items2 := []services.IOSSystemContextMenuItemData{services.IOSSystemContextMenuItemDataCut{}}
		controller.ShowWithItems(rect, items2)
		expect(controller.IsVisible() && len(*received) == 2 && itemListsEqual((*received)[1], items2), "%v", *received)
		controller.Hide()
		expect(!controller.IsVisible(), "visible")
	})
}

func TestShowingAControllerWithEmptyItems(t *testing.T) {
	contextMenuTest(t, func() {
		_, received, _ := recordMenuCalls()
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		mustPanicAssertion(func() { controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), nil) })
		expect(!controller.IsVisible() && len(*received) == 0, "shown")
	})
}

func TestCustomActionCallbacksAreProperlyManaged(t *testing.T) {
	contextMenuTest(t, func() {
		action1Called, action2Called := false, false
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		item1 := services.NewIOSSystemContextMenuItemDataCustom("Action 1", func() { action1Called = true })
		item2 := services.NewIOSSystemContextMenuItemDataCustom("Action 2", func() { action2Called = true })
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{item1, item2})
		expect(controller.IsVisible(), "not visible")
		controller.HandleCustomContextMenuAction(item1.CallbackID())
		expect(action1Called && !action2Called, "actions")
		controller.HandleCustomContextMenuAction(item2.CallbackID())
		expect(action1Called && action2Called, "actions")
		controller.Hide()
		expect(!controller.IsVisible(), "visible")
		mustPanicAssertion(func() { controller.HandleCustomContextMenuAction(item1.CallbackID()) })
	})
}

func TestMultipleControllersHandleCallbacksIndependently(t *testing.T) {
	contextMenuTest(t, func() {
		called1, called2 := false, false
		controller1 := services.NewSystemContextMenuController()
		controller2 := services.NewSystemContextMenuController()
		defer func() {
			controller1.Dispose()
			controller2.Dispose()
		}()
		item1 := services.NewIOSSystemContextMenuItemDataCustom("Controller 1 Action", func() { called1 = true })
		item2 := services.NewIOSSystemContextMenuItemDataCustom("Controller 2 Action", func() { called2 = true })
		controller1.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{item1})
		expect(controller1.IsVisible(), "1 not visible")
		controller1.Hide()
		expect(!controller1.IsVisible(), "1 visible")
		controller2.ShowWithItems(ui.NewRectFromLTWH(100, 100, 100, 100), []services.IOSSystemContextMenuItemData{item2})
		expect(controller2.IsVisible(), "2 not visible")
		controller2.HandleCustomContextMenuAction(item2.CallbackID())
		expect(called2 && !called1, "actions")
		mustPanicAssertion(func() { controller1.HandleCustomContextMenuAction(item1.CallbackID()) })
	})
}

func TestPlatformDismissalClearsCallbacks(t *testing.T) {
	contextMenuTest(t, func() {
		called := false
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		item := services.NewIOSSystemContextMenuItemDataCustom("Test Action", func() { called = true })
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{item})
		expect(controller.IsVisible(), "not visible")
		controller.HandleSystemHide()
		expect(!controller.IsVisible(), "visible")
		mustPanicAssertion(func() { controller.HandleCustomContextMenuAction(item.CallbackID()) })
		expect(!called, "called")
	})
}

func TestHandleCustomContextMenuActionWithNoSystemContextMenuClient(t *testing.T) {
	ensureInitialized()
	services.ServicesBindingSetSystemContextMenuClient(nil)
	runAsync(t, func() {
		message := jsonMethod.EncodeMethodCall(services.MethodCall{Method: "ContextMenu.onPerformCustomAction", Arguments: []any{0, "test-id"}})
		// returnsNormally: the binding's assertion becomes an error reply.
		messenger().HandlePlatformMessage("flutter/platform", message, func([]byte) {}).Await()
	})
}

func TestHandleCustomContextMenuActionWithNonExistentCallbackID(t *testing.T) {
	contextMenuTest(t, func() {
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{
			services.NewIOSSystemContextMenuItemDataCustom("Test Action", func() {}),
		})
		expect(controller.IsVisible(), "not visible")
		mustPanicAssertion(func() { controller.HandleCustomContextMenuAction("non-existent-id") })
	})
}

func TestHandleCustomContextMenuActionAfterHideClearsCallbacks(t *testing.T) {
	contextMenuTest(t, func() {
		called := false
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		item := services.NewIOSSystemContextMenuItemDataCustom("Test Action", func() { called = true })
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{item})
		expect(controller.IsVisible(), "not visible")
		controller.HandleCustomContextMenuAction(item.CallbackID())
		expect(called, "not called")
		called = false
		controller.Hide()
		expect(!controller.IsVisible(), "visible")
		mustPanicAssertion(func() { controller.HandleCustomContextMenuAction(item.CallbackID()) })
		expect(!called, "called")
	})
}

func TestShowingNewMenuInvalidatesOldMenuCallbacks(t *testing.T) {
	contextMenuTest(t, func() {
		oldCalled, newCalled := false, false
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		oldItem := services.NewIOSSystemContextMenuItemDataCustom("Old Action", func() { oldCalled = true })
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 100, 100), []services.IOSSystemContextMenuItemData{oldItem})
		expect(controller.IsVisible(), "not visible")
		newItem := services.NewIOSSystemContextMenuItemDataCustom("New Action", func() { newCalled = true })
		controller.ShowWithItems(ui.NewRectFromLTWH(100, 100, 100, 100), []services.IOSSystemContextMenuItemData{newItem})
		expect(controller.IsVisible(), "not visible")
		mustPanicAssertion(func() { controller.HandleCustomContextMenuAction(oldItem.CallbackID()) })
		expect(!oldCalled, "old called")
		controller.HandleCustomContextMenuAction(newItem.CallbackID())
		expect(newCalled && !oldCalled, "actions")
	})
}

// Own: a system hide is recorded (_hiddenBySystem), and the next show
// clears it.
func TestSystemHideIsRecorded(t *testing.T) {
	contextMenuTest(t, func() {
		controller := services.NewSystemContextMenuController()
		defer controller.Dispose()
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 1, 1), copyItems)
		controller.HandleSystemHide()
		expect(nodeValue(propertyNodes(controller)[2]) == true, "_hiddenBySystem %v", nodeValue(propertyNodes(controller)[2]))
		controller.ShowWithItems(ui.NewRectFromLTWH(0, 0, 1, 1), copyItems)
		expect(nodeValue(propertyNodes(controller)[2]) == false && controller.IsVisible(), "shown again")
		controller.Hide()
	})
}
