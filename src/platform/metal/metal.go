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
#import <objc/runtime.h>

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

// Impeller commits its command buffers on its own queue, which the C API
// doesn't expose, and Metal doesn't order command buffers across queues.
// To know when a frame drawn offscreen is finished, the commits made while
// fg_render_offscreen runs are recorded (the command buffer class's
// `commit` is wrapped once) and waited for before the pixels are read.
static IMP fg_orig_commit = NULL;
static NSMutableArray* fg_capture = nil;

static void fg_commit(id self, SEL _cmd) {
  if (fg_capture != nil) [fg_capture addObject:self];
  ((void (*)(id, SEL))fg_orig_commit)(self, _cmd);
}

static void fg_wrap_commit(id<MTLCommandQueue> queue) {
  id<MTLCommandBuffer> probe = [queue commandBuffer];
  Method m = class_getInstanceMethod([probe class], @selector(commit));
  // Override on the concrete class (not a superclass shared by others).
  if (!class_addMethod([probe class], @selector(commit), (IMP)fg_commit, method_getTypeEncoding(m))) {
    fg_orig_commit = method_setImplementation(m, (IMP)fg_commit);
  } else {
    fg_orig_commit = method_getImplementation(m);
  }
}

// fg_render_offscreen draws a display list into a w×h BGRA8 drawable of a
// layer that is not on screen, waits for Impeller's command buffers, and
// copies the pixels to out (w*h*4 bytes, BGRA, premultiplied).
static int fg_render_offscreen(ImpellerContext ctx, ImpellerDisplayList dl, int w, int h, void* out) {
  @autoreleasepool {
    static CAMetalLayer* layer = nil;
    static id<MTLCommandQueue> queue = nil;
    if (layer == nil) {
      layer = [CAMetalLayer layer];
      layer.framebufferOnly = NO;
      layer.pixelFormat = MTLPixelFormatBGRA8Unorm;
      layer.device = MTLCreateSystemDefaultDevice();
      queue = [layer.device newCommandQueue];
      fg_wrap_commit(queue);
    }
    layer.drawableSize = CGSizeMake(w, h);
    id<CAMetalDrawable> drawable = [layer nextDrawable];
    if (drawable == nil) return 0;
    ImpellerSurface surface =
        ImpellerSurfaceCreateWrappedMetalDrawableNew(ctx, (__bridge void*)drawable);
    if (surface == NULL) return 0;
    fg_capture = [NSMutableArray array];
    bool ok = ImpellerSurfaceDrawDisplayList(surface, dl);
    NSArray* committed = fg_capture;
    fg_capture = nil;
    ImpellerSurfaceRelease(surface);
    for (id<MTLCommandBuffer> cb in committed) [cb waitUntilCompleted];
    if (!ok) return 0;
    id<MTLTexture> tex = drawable.texture;
    NSUInteger bpr = (NSUInteger)w * 4;
    id<MTLBuffer> buf = [layer.device newBufferWithLength:bpr * h options:MTLResourceStorageModeShared];
    id<MTLCommandBuffer> cb = [queue commandBuffer];
    id<MTLBlitCommandEncoder> blit = [cb blitCommandEncoder];
    [blit copyFromTexture:tex sourceSlice:0 sourceLevel:0 sourceOrigin:MTLOriginMake(0, 0, 0)
               sourceSize:MTLSizeMake(w, h, 1) toBuffer:buf destinationOffset:0
      destinationBytesPerRow:bpr destinationBytesPerImage:bpr * h];
    [blit endEncoding];
    [cb commit];
    [cb waitUntilCompleted];
    memcpy(out, buf.contents, bpr * h);
    return 1;
  }
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

var offscreenMu sync.Mutex

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

// RenderOffscreen draws a display list (impeller.DisplayList.Handle) with an
// Impeller context (impeller.Context.Handle) into a width×height target that
// is never shown, and returns its pixels as premultiplied RGBA, row by row.
func RenderOffscreen(ctx, displayList unsafe.Pointer, width, height int) ([]byte, error) {
	if width <= 0 || height <= 0 {
		return nil, errors.New("metal: empty offscreen target")
	}
	offscreenMu.Lock()
	defer offscreenMu.Unlock()
	out := make([]byte, width*height*4)
	if C.fg_render_offscreen(C.ImpellerContext(ctx), C.ImpellerDisplayList(displayList), C.int(width), C.int(height), unsafe.Pointer(&out[0])) == 0 {
		return nil, errors.New("metal: offscreen render failed")
	}
	for i := 0; i < len(out); i += 4 { // BGRA → RGBA
		out[i], out[i+2] = out[i+2], out[i]
	}
	return out, nil
}
