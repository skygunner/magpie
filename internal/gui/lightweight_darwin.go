//go:build !nogui

package gui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#include <stdlib.h>

// screenWorkAreas gives every display's work area in the logical points
// Wails uses for a window's absolute position: from the primary screen's
// upper left, y down. AppKit's own origin is the lower left, y up.
static NSRect *screenWorkAreas(NSUInteger *count) {
	NSArray<NSScreen *> *screens = [NSScreen screens];
	NSUInteger n = screens.count;
	*count = n;
	if (n == 0) return NULL;
	NSRect *rects = malloc(n * sizeof(NSRect));
	if (rects == NULL) {
		*count = 0;
		return NULL;
	}
	CGFloat top = NSMaxY(screens.firstObject.frame);
	for (NSUInteger i = 0; i < n; i++) {
		NSRect rect = screens[i].visibleFrame;
		rect.origin.y = top - NSMaxY(rect);
		rects[i] = rect;
	}
	return rects;
}

static WKWebView *findWebView(NSView *v) {
	if ([v isKindOfClass:[WKWebView class]]) return (WKWebView *)v;
	for (NSView *s in v.subviews) {
		WKWebView *w = findWebView(s);
		if (w) return w;
	}
	return nil;
}

// dropWebView takes the window's webview out of it, on the main thread: a
// Wails window closed keeps its WKWebView, and its web content process,
// alive (seen: both pages' processes, 65 MB each, still there minutes
// after). The window's own pointer to it is cleared first (Wails' dealloc
// reads it), then what holds the webview: its delegates, its script
// handlers, its superview; it is freed and its process ends.
static void dropWebView(void *win) {
	NSWindow *w = (NSWindow *)win;
	WKWebView *web = findWebView(w.contentView);
	if (!web) return;
	if ([w respondsToSelector:@selector(setWebView:)]) [w performSelector:@selector(setWebView:) withObject:nil];
	[web stopLoading];
	web.navigationDelegate = nil;
	web.UIDelegate = nil;
	[web.configuration.userContentController removeAllScriptMessageHandlers];
	[web removeFromSuperview];
}
*/
import "C"

import (
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// nativeWorkAreas reads AppKit's work areas, for the first show that comes
// before Wails' screen cache (#2785); on the main thread, as NSScreen asks.
func nativeWorkAreas() []area {
	var count C.NSUInteger
	rects := C.screenWorkAreas(&count)
	if rects == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(rects))
	out := make([]area, 0, int(count))
	for _, rect := range unsafe.Slice(rects, int(count)) {
		if width, height := int(rect.size.width), int(rect.size.height); width > 0 && height > 0 {
			out = append(out, area{int(rect.origin.x), int(rect.origin.y), width, height})
		}
	}
	return out
}

// dropWebView ends w's webview before it is closed; on the main thread.
func dropWebView(w *application.WebviewWindow) {
	if p := w.NativeWindow(); p != nil {
		C.dropWebView(p)
	}
}
