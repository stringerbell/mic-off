//go:build darwin

#include "_cgo_export.h"
#import <Carbon/Carbon.h>
#import <Cocoa/Cocoa.h>

static EventHandlerRef handlerRef = NULL;

static OSStatus pressedHandler(EventHandlerCallRef next, EventRef ev, void *userData) {
	EventHotKeyID k;
	GetEventParameter(ev, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(k), NULL, &k);
	micoffHotkeyPressed(k.id);
	return noErr;
}

static void ensureHandler(void) {
	if (handlerRef != NULL) return;
	EventTypeSpec spec = { kEventClassKeyboard, kEventHotKeyPressed };
	InstallApplicationEventHandler(&pressedHandler, 1, &spec, NULL, &handlerRef);
}

// Carbon hotkey calls must happen on the main thread.
static void onMain(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_sync(dispatch_get_main_queue(), block);
	}
}

int micoff_register(uint32_t key, uint32_t mods, uint32_t id, EventHotKeyRef *ref) {
	__block OSStatus st = noErr;
	onMain(^{
		ensureHandler();
		EventHotKeyID hk;
		hk.signature = 0x4D4F4646; // 'MOFF'
		hk.id = id;
		st = RegisterEventHotKey(key, mods, hk, GetApplicationEventTarget(), 0, ref);
	});
	return (int)st;
}

int micoff_unregister(EventHotKeyRef ref) {
	__block OSStatus st = noErr;
	onMain(^{ st = UnregisterEventHotKey(ref); });
	return (int)st;
}
