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
