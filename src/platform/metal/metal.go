// Package metal attaches a CAMetalLayer to a GLFW Cocoa window and presents
// Impeller display lists to it. This is the only Objective-C in the project.
package metal

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -I${SRCDIR}/../../third_party/impeller_sdk/include
#cgo LDFLAGS: -framework AppKit -framework Metal -framework QuartzCore
#include "impeller.h"
#import <AppKit/AppKit.h>
#import <Metal/Metal.h>
#import <QuartzCore/QuartzCore.h>

static void* fg_attach_layer(void* nswindow) {
  NSWindow* window = (__bridge NSWindow*)nswindow;
  CAMetalLayer* layer = [CAMetalLayer layer];
  layer.framebufferOnly = NO;
  layer.pixelFormat = MTLPixelFormatBGRA8Unorm;
  layer.device = MTLCreateSystemDefaultDevice();
  layer.contentsScale = window.backingScaleFactor;
  window.contentView.layer = layer;
  window.contentView.wantsLayer = YES;
  return (__bridge_retained void*)layer;
}

static int fg_present(void* layer_ptr, ImpellerContext ctx, ImpellerDisplayList dl) {
  @autoreleasepool {
    CAMetalLayer* layer = (__bridge CAMetalLayer*)layer_ptr;
    CGSize size = layer.bounds.size;
    CGFloat scale = layer.contentsScale;
    layer.drawableSize = CGSizeMake(size.width * scale, size.height * scale);
    id<CAMetalDrawable> drawable = [layer nextDrawable];
    if (drawable == nil) return 0;
    ImpellerSurface surface =
        ImpellerSurfaceCreateWrappedMetalDrawableNew(ctx, (__bridge void*)drawable);
    if (surface == NULL) return 0;
    bool ok = ImpellerSurfaceDrawDisplayList(surface, dl) && ImpellerSurfacePresent(surface);
    ImpellerSurfaceRelease(surface);
    return ok ? 1 : 0;
  }
}

static double fg_scale(void* layer_ptr) {
  return ((__bridge CAMetalLayer*)layer_ptr).contentsScale;
}

static long fg_window_number(void* nswindow) {
  return ((__bridge NSWindow*)nswindow).windowNumber;
}

static void fg_release_layer(void* layer_ptr) { CFBridgingRelease(layer_ptr); }
*/
import "C"

import (
	"errors"
	"unsafe"
)

type Layer struct{ p unsafe.Pointer }

// Attach installs a CAMetalLayer as the content of an NSWindow*.
func Attach(nsWindow unsafe.Pointer) *Layer { return &Layer{C.fg_attach_layer(nsWindow)} }

// Scale is the backing scale factor (2.0 on Retina displays).
func (l *Layer) Scale() float32 { return float32(C.fg_scale(l.p)) }

// Present draws a display list (impeller.DisplayList.Handle) using an
// Impeller context (impeller.Context.Handle) and shows it on screen.
func (l *Layer) Present(ctx, displayList unsafe.Pointer) error {
	if C.fg_present(l.p, C.ImpellerContext(ctx), C.ImpellerDisplayList(displayList)) == 0 {
		return errors.New("metal: present failed")
	}
	return nil
}

// WindowNumber returns the macOS window-server ID of an NSWindow*
// (usable with `screencapture -l`).
func WindowNumber(nsWindow unsafe.Pointer) int { return int(C.fg_window_number(nsWindow)) }

func (l *Layer) Release() { C.fg_release_layer(l.p) }
