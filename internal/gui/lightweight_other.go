//go:build !darwin && !nogui

package gui

import "github.com/wailsapp/wails/v3/pkg/application"

// nativeWorkAreas: off the Mac Wails' screen cache is the only list, and
// nothing reads it early enough to need another.
func nativeWorkAreas() []area { return nil }

// dropWebView: closing a window ends its webview off the Mac.
func dropWebView(*application.WebviewWindow) {}
