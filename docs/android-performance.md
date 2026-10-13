# Android frame work — 2026-10-13

Gio already renders through Android OpenGL ES, with Vulkan as another available
backend (`third_party/gio/app/os_android.go`). The main measured bottleneck was
CPU work before those GPU operations: `Localizer.Text` cloned the entire gettext
catalog for every label, every frame. It now builds immutable string maps once
at initialization and performs a direct lookup. Language switching and fallback
remain covered by tests. Each unformatted lookup allocates zero objects.

Phone status cards, the Chats menu, context menus and the emoji/sticker panel now
reuse recorded rounded clips through the bounded shape cache. This lets Gio
reuse their path data instead of tessellating and uploading new buffers every
frame. Idle tests now cover Android layouts at 2x density as well as desktop:
settled animations request no immediate new frame (editor blink may schedule a
later frame). Decoded image rendering fields are published as immutable replacement entries
under the cache lock, eliminating the frame/decoder race detected during final
checks. Cache hits retain their existing pointer lookup without snapshot copies.
Image decoding remains asynchronous with the existing decode
semaphore and byte-bounded image cache; no tighter Go heap limit is introduced.

## CPU measurements

Go 1.27.2, Linux amd64, AMD EPYC 9V74, 8 Go workers. The UI benchmark uses the
same reference mock backend, 360×720 dp at 2x density, and warmed frames. It
includes layout, event updates and router processing; it excludes GPU execution,
presentation, network delivery and physical Android scheduling. Baseline is one
1-second run of the previous i18n implementation. After is the median of three
1-second runs, taken without another build/test process running.

| View | Before ms/frame | After ms/frame | Before bytes/frame | After bytes/frame | Before allocations/frame | After allocations/frame |
|---|---:|---:|---:|---:|---:|---:|
| Chats | 4.135 | 0.253 | 5,009,052 | 6,600 | 40,051 | 191 |
| Conversation | 1.124 | 0.266 | 1,013,962 | 13,467 | 8,254 | 282 |
| Updates | 4.513 | 0.217 | 5,512,804 | 8,708 | 44,052 | 201 |

These are CPU/layout improvements, not a claim of 17–20× Android FPS. GPU frame
time, slow frames, battery use and long-session memory must still be measured on
a booted Android target. No physical device or paired account was available.

Run:

```sh
go test ./internal/i18n -bench '^BenchmarkText$' -run '^TestTextDoesNotAllocatePerLabel$'
go test ./internal/ui -run '^$' -bench '^BenchmarkMobileFrame$' -benchtime=1s -count=3
go test ./internal/ui -run '^TestIdleAtRest$'
```

## Raw results

Baseline:

```text
goos: linux
goarch: amd64
pkg: github.com/muleff/WaGo/internal/ui
cpu: AMD EPYC 9V74 80-Core Processor                
BenchmarkMobileFrame/chats-8         	     283	   4134818 ns/op	 5009052 B/op	   40051 allocs/op
BenchmarkMobileFrame/conversation-8  	    1159	   1124044 ns/op	 1013962 B/op	    8254 allocs/op
BenchmarkMobileFrame/updates-8       	     273	   4513334 ns/op	 5512804 B/op	   44052 allocs/op
PASS
ok  	github.com/muleff/WaGo/internal/ui	4.379s
```

After:

```text
goos: linux
goarch: amd64
pkg: github.com/muleff/WaGo/internal/ui
cpu: AMD EPYC 9V74 80-Core Processor                
BenchmarkMobileFrame/chats-8         	    4819	    238036 ns/op	    6600 B/op	     191 allocs/op
BenchmarkMobileFrame/chats-8         	    5107	    253192 ns/op	    6600 B/op	     191 allocs/op
BenchmarkMobileFrame/chats-8         	    4701	    262069 ns/op	    6600 B/op	     191 allocs/op
BenchmarkMobileFrame/conversation-8  	    4270	    287992 ns/op	   13467 B/op	     282 allocs/op
BenchmarkMobileFrame/conversation-8  	    4592	    265720 ns/op	   13466 B/op	     282 allocs/op
BenchmarkMobileFrame/conversation-8  	    4502	    264484 ns/op	   13467 B/op	     282 allocs/op
BenchmarkMobileFrame/updates-8       	    5371	    212713 ns/op	    8707 B/op	     201 allocs/op
BenchmarkMobileFrame/updates-8       	    5084	    220429 ns/op	    8708 B/op	     201 allocs/op
BenchmarkMobileFrame/updates-8       	    5392	    217275 ns/op	    8708 B/op	     201 allocs/op
PASS
ok  	github.com/muleff/WaGo/internal/ui	11.031s
```
