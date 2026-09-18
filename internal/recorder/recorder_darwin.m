//go:build darwin

// The native "Change hotkey" window. All logic lives in Go (session.go);
// this file only draws the window, forwards key events, and shows text.

#include "_cgo_export.h"
#import <Cocoa/Cocoa.h>

static NSWindow *win = nil;
static NSTextField *comboLabel = nil;
static NSTextField *hintLabel = nil;
static NSButton *saveButton = nil;
static id controller = nil;
static BOOL modalRunning = NO;

// Runs block on the main thread and waits. Uses a run-loop block rather
// than the main dispatch queue so it also works while a modal window is up.
static void onMain(void (^block)(void)) {
	if ([NSThread isMainThread]) {
		block();
		return;
	}
	dispatch_semaphore_t done = dispatch_semaphore_create(0);
	CFRunLoopPerformBlock(CFRunLoopGetMain(), kCFRunLoopCommonModes, ^{
		block();
		dispatch_semaphore_signal(done);
	});
	CFRunLoopWakeUp(CFRunLoopGetMain());
	dispatch_semaphore_wait(done, DISPATCH_TIME_FOREVER);
	dispatch_release(done);
}

static void stopModal(void) {
	if (modalRunning) {
		modalRunning = NO;
		[NSApp stopModal];
	}
}

// The content view owns every key press. Even ⌘-combinations are taken
// here (performKeyEquivalent:) so nothing reaches a menu shortcut such as ⌘Q.
@interface MicoffRecorderView : NSView
@end

@implementation MicoffRecorderView
- (BOOL)acceptsFirstResponder { return YES; }
- (void)keyDown:(NSEvent *)e {
	if ([e isARepeat]) return;
	micoffRecorderKeyDown((uint32_t)e.keyCode, (uint32_t)e.modifierFlags);
}
- (void)keyUp:(NSEvent *)e {}
- (void)flagsChanged:(NSEvent *)e {
	micoffRecorderFlags((uint32_t)e.modifierFlags);
}
- (BOOL)performKeyEquivalent:(NSEvent *)e {
	if (e.type == NSEventTypeKeyDown) [self keyDown:e];
	return YES;
}
@end

@interface MicoffRecorderController : NSObject <NSWindowDelegate>
@end

@implementation MicoffRecorderController
- (void)save:(id)sender { micoffRecorderSave(); }
- (void)cancel:(id)sender { micoffRecorderCancel(); }
- (void)windowWillClose:(NSNotification *)n { stopModal(); }
@end

static NSTextField *makeLabel(NSView *parent, NSRect frame, NSFont *font, NSColor *color) {
	NSTextField *l = [[NSTextField alloc] initWithFrame:frame];
	l.editable = NO;
	l.selectable = NO;
	l.bordered = NO;
	l.drawsBackground = NO;
	l.alignment = NSTextAlignmentCenter;
	l.font = font;
	l.textColor = color;
	l.lineBreakMode = NSLineBreakByWordWrapping;
	[[l cell] setWraps:YES];
	[parent addSubview:l];
	[l release];
	return l;
}

static NSButton *makeButton(NSView *parent, NSRect frame, NSString *title, SEL action) {
	NSButton *b = [[NSButton alloc] initWithFrame:frame];
	b.title = title;
	b.bezelStyle = NSBezelStyleRounded;
	b.target = controller;
	b.action = action;
	b.refusesFirstResponder = YES; // key presses always go to the recorder view
	[parent addSubview:b];
	[b release];
	return b;
}

static void build(NSString *combo, NSString *hint, BOOL canSave) {
	NSRect frame = NSMakeRect(0, 0, 420, 184);
	win = [[NSWindow alloc] initWithContentRect:frame
	                                  styleMask:(NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
	                                    backing:NSBackingStoreBuffered
	                                      defer:NO];
	win.title = @"Change hotkey";
	win.releasedWhenClosed = NO;
	controller = [[MicoffRecorderController alloc] init];
	win.delegate = controller;

	MicoffRecorderView *view = [[MicoffRecorderView alloc] initWithFrame:frame];
	win.contentView = view;
	[view release];

	comboLabel = makeLabel(view, NSMakeRect(20, 104, 380, 48),
	                       [NSFont systemFontOfSize:34 weight:NSFontWeightSemibold],
	                       [NSColor labelColor]);
	hintLabel = makeLabel(view, NSMakeRect(20, 60, 380, 36),
	                      [NSFont systemFontOfSize:12],
	                      [NSColor secondaryLabelColor]);
	makeButton(view, NSMakeRect(222, 14, 90, 32), @"Cancel", @selector(cancel:));
	saveButton = makeButton(view, NSMakeRect(316, 14, 90, 32), @"Save", @selector(save:));
	saveButton.keyEquivalent = @"\r"; // draws as the default (blue) button

	comboLabel.stringValue = combo;
	hintLabel.stringValue = hint;
	saveButton.enabled = canSave;
	[win center];
}

void micoff_recorder_run(const char *combo, const char *hint, int canSave) {
	onMain(^{
		build([NSString stringWithUTF8String:combo], [NSString stringWithUTF8String:hint], canSave != 0);
		[NSApp activateIgnoringOtherApps:YES];
		[win makeKeyAndOrderFront:nil];
		[win makeFirstResponder:win.contentView];
		modalRunning = YES;
		[NSApp runModalForWindow:win];
		modalRunning = NO;
		[win orderOut:nil];
		win.delegate = nil;
		comboLabel = nil;
		hintLabel = nil;
		saveButton = nil;
		[win release];
		win = nil;
		[controller release];
		controller = nil;
	});
}

void micoff_recorder_set(const char *combo, const char *hint, int canSave) {
	NSString *c = [NSString stringWithUTF8String:combo];
	NSString *h = [NSString stringWithUTF8String:hint];
	onMain(^{
		if (win == nil) return;
		comboLabel.stringValue = c;
		hintLabel.stringValue = h;
		saveButton.enabled = canSave != 0;
	});
}

void micoff_recorder_close(void) {
	onMain(^{
		if (win == nil) return;
		stopModal();
		[win close];
	});
}
