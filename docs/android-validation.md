# Android validation — 2026-10-13

## Changes

Native SAF file selection preserves provider filenames and MIME types. Private
copies remain readable by asynchronous uploads through held file descriptors.
The non-exported content provider opens existing private media through exact URI
grants, without making duplicate share copies.

Microphone permission is requested on use. TMPDIR points to app-private storage.
Capture stops before draining and closing Opus. Playback starts on the first tap,
counts actual PCM consumption and handles pause, seek and close responsively.

Mobile navigation has Chats, Novedades (states and channels), Comunidades and
Llamadas. Contact details occupy the full width with one back button. Text fonts
precede emoji fonts, fixing phone digit spacing. Spanish copy covers contact,
linking, recording and calendar UI. Empty gettext keys cannot expose headers.

Brief backgrounding preserves the backend. A visible foreground service uses
remoteMessaging on Android 14+ and dataSync on earlier releases. Its initial
linking wake lock is bounded to ten minutes. Foreground recovery retries the same
client without resetting phone-pairing state. Android uses the permanent client
event handler for QR/pairing events. Unpaired disconnects remain in login UI;
offline send attempts preserve drafts.

## Executed checks

- Go 1.27.2: `go test -json -count=1 ./...`: 326 passed, 0 failed, 1 skipped.
  The optional screenshot test was run separately and passed.
- `go vet ./...`, `go build ./...`, gofmt and `git diff --check`: passed.
- Android Java sources compiled against API 35 using Java 17.
- Software EGL rendered 320 dp, 390 dp, landscape, keyboard-sized and enlarged
  text variants of chats, contact, Novedades and login. Evidence is in
  `docs/screenshots/android-*.png`; these are renders, not device screenshots.
- APK includes ARMv7 and ARM64, minimum SDK 21, target SDK 35, application ID
  `com.muleff.wago`, development version `0.1.0.7`.
- apksigner verified JAR/v2/v3 signatures. aapt verified manifest, provider and
  foreground service. Both native ABI libraries are present.

## Limits

No physical Android device or paired WhatsApp session was available. A software
emulator without KVM did not reach an accessible booted adb state. Native
permission, picker, service and microphone flows therefore have build/source/
regression coverage, not device-runtime proof. Real pairing, delivery and
reception still require an authorized account and recipient.

WhatsApp voice/video call signaling remains unsupported by the existing backend.
Camera capture is not added. The APK uses a development signature; compatibility
with an existing installation's signing key has not been verified.

## GitHub workflows

The latest inspected main run succeeded:
https://github.com/muleff/WaGo/actions/runs/38337603738
It built commit `957b47d1dad40ba6189612ff48275ebd0fe2855e`, before this work.
The updated workflow installs patched gogio and does not demand global storage
permission for SAF. Changes are on local branch `fix/android-functional`.
A remote run for these changes requires publishing that branch; the old main
run does not validate this APK.

## Platform references

- Pinned Gio/gogio v0.10.3 source and permission documentation.
- https://developer.android.com/training/data-storage/shared/documents-files
- https://developer.android.com/training/permissions/requesting
- https://developer.android.com/develop/background-work/services/fgs/service-types

## Mobile layout and swipe follow-up

Removed desktop spacing from secondary mobile pages and reduced direct chat
margins to 8 dp. Group chats retain room for sender avatars. Settings categories
and the new chat flow use the full available height. Login no longer wraps its
form in a second bordered card. New-chat and community actions are translated.

System inset pixels are painted with the panel color and the Android status and
navigation bar colors follow the theme. Interactive controls remain in the safe
area; system icons are not hidden by an immersive/fullscreen mode.

Touch swipes to the right move the message, show a reply indicator and quote the
message on release after 56 dp. The message animates back into position. Short,
left, vertical and cancelled drags do not reply. A real Gio list regression
confirms vertical touch scrolling continues to work.

Verified: complete Go tests, vet, build and formatting; rendered 12 mobile views
at 320/390 dp, landscape, keyboard height and large system text (60 renders).
The headless inset regression reproduced transparent system-bar backgrounds
before the fix and verifies panel-colored pixels afterwards. ARM/ARM64 APK
built locally and its v1/v2/v3 signatures verified. Device-runtime status-bar
appearance and touch feel still need checking on a physical Android device.
The previous remote Android job succeeded; desktop jobs failed only on two
unformatted fields in model.Attachment, now formatted for the next run.

## Reference-based mobile correction

The earlier spacing fix missed chat/channel rows, whose nested desktop insets
still wasted horizontal space. Mobile rows now extend to both screen edges,
retain only content padding, and use stronger name typography. Filter chips
scroll horizontally. The mobile navigation highlights the active destination;
chat creation and status posting use floating action buttons. Novedades uses
horizontal status preview cards above channel rows. The mobile composer keeps
its microphone/send circle separate from the message pill. Group conversations
no longer reserve both side gutters for desktop sender avatars. Mobile colors
and wallpaper scale were checked against the supplied visual references while
retaining WaGo's identity and supported features.

Presence dates now compare calendar days in the local timezone, including month
and year boundaries. Android reads java.util.TimeZone before starting the
backend and bundles zone data. Presence/typing copy is localized; mobile clocks
use Spanish 12-hour notation. Generated media, deletion, editing, status and
menu labels have additional Spanish coverage. Contact names and message/caption
content remain unchanged, with regressions covering names and captions that
match translatable UI strings.

Verification: complete Go suite, vet, build and formatting; 70 headless renders
cover 14 views at 320/390 dp, landscape, keyboard height and large text. Tests
cover full-width row taps, horizontal status scrolling, vertical message
scrolling, swipe-to-reply, Spanish presence and media copy, timezone loading and
last-seen calendar boundaries. Actual phone appearance and touch interactions
still need device testing; headless renders do not provide that evidence.


## Mobile menu and performance correction

The mobile Chats menu now has a screen-relative anchor, bounded width/position,
and scrolling when the viewport is short. Touch regressions exercise the actual
three-dot header button and New group at 1x and 2x density. New group, New
community, Starred messages, Mark all as read and Settings use real application
flows. New community sends hypermeow's `ReqCreateGroup.IsParent` and refreshes
linked groups. Its creation flow is covered with the mock backend; real-account
server creation still needs device/account testing.

Linked devices remains omitted. Broadcast lists opens an explicit capability
notice: pinned hypermeow rejects non-status broadcast sending with
`ErrBroadcastListUnsupported`. No fake list or silent per-contact send replaces
a native broadcast list.

Mobile attachments use a bounded action grid above the composer, with existing
document/photo/audio/contact/poll flows. The Contact cell has a touch regression.
The supported actions are translated to Spanish. Large avatars/icons now reserve
a measured 12dp gap before text; member badges, your member row and sticker
empty states are translated. Group stickers regain a small sender avatar and a
compact name header, without reintroducing large gutters for every message.
The emoji/sticker panel uses the phone width and a bounded height.

The per-label catalog copy is removed and rounded GPU clips are reused. Before /
after CPU measurements and limitations are in `android-performance.md`. Fresh
headless screenshots cover 19 views across 320/390dp, landscape, keyboard height,
large text and 360dp at 2x density (114 renders). They do not prove physical-device
FPS or live WhatsApp delivery.

Final checks for this correction: complete Go test run — 358 passed, 0 failed,
2 skipped (optional headless tests without their environment flag); `go vet ./...`,
`go build ./...`, gofmt and `git diff --check` passed. The 114-render headless
run was executed separately and passed. APK CI status is reported with its
GitHub Actions run in the delivery message.

The additional race check reproduced concurrent mutation of decoded image
entries while frames read them. Decoders now publish a replacement entry with
immutable rendering fields under the cache lock. This preserves asynchronous
decoding and cache accounting without copying an entry on every cache hit.
Regressions verify immutable snapshots and no resurrection of forgotten loads.

The focused mobile/image/i18n race suite passed with zero detected races. A
full UI race run also detected zero races but failed `TestClickRedraws` once
(the raw-op comparison observed one extra frame); its isolated repeat passed.
That initial full race run is not counted as a passing suite.

A fresh complete `go test -race ./internal/ui -count=1`, run without other
CPU-heavy checks concurrently, passed (66.172 seconds) with zero detected races.
