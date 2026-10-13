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
