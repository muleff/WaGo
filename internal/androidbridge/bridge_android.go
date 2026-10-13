//go:build android

package androidbridge

/*
#include <jni.h>
#include <stdint.h>
char *wago_bytes(JNIEnv *env, jbyteArray bytes, int *len);
#include <stdlib.h>
int wago_request(uintptr_t vm, uintptr_t context, long long id, int kind, const char *payload);
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"gioui.org/app"
	"os"
	"sync"
	"time"
	"unsafe"
)

var requests requestPool

func request(ctx context.Context, kind int, payload string) ([]PickedFile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, ch := requests.begin()
	s := C.CString(payload)
	defer C.free(unsafe.Pointer(s))
	if C.wago_request(C.uintptr_t(app.JavaVM()), C.uintptr_t(app.AppContext()), C.longlong(id), C.int(kind), s) != 0 {
		requests.complete(id, result{err: ErrActivityUnavailable})
	}
	return requests.wait(ctx, id, ch)
}
func RequestMicrophone(ctx context.Context) error { _, err := request(ctx, 0, ""); return err }
func PickFiles(ctx context.Context, multiple bool, mimeTypes []string) ([]PickedFile, error) {
	payload, _ := json.Marshal(struct {
		Multiple bool     `json:"multiple"`
		MIME     []string `json:"mime"`
	}{multiple, mimeTypes})
	return request(ctx, 1, string(payload))
}
func OpenMedia(path, mime string) error {
	payload, _ := json.Marshal(struct {
		Path string `json:"path"`
		MIME string `json:"mime"`
	}{path, mime})
	_, err := request(context.Background(), 2, string(payload))
	return err
}

//export Java_org_gioui_WaGoBridge_nativeResult
func Java_org_gioui_WaGoBridge_nativeResult(env *C.JNIEnv, cls C.jclass, id C.jlong, payload C.jbyteArray) {
	var n C.int
	raw := C.wago_bytes(env, payload, &n)
	if raw == nil {
		requests.complete(uint64(id), result{err: ErrActivityUnavailable})
		return
	}
	defer C.free(unsafe.Pointer(raw))
	var decoded struct {
		Files []PickedFile `json:"files"`
		Error string       `json:"error"`
	}
	err := json.Unmarshal(C.GoBytes(unsafe.Pointer(raw), n), &decoded)
	if err == nil && decoded.Error != "" {
		if decoded.Error == "permission" {
			err = ErrPermissionDenied
		} else {
			err = errors.New(decoded.Error)
		}
	}
	if !requests.complete(uint64(id), result{files: decoded.Files, err: err}) {
		for _, f := range decoded.Files {
			_ = os.Remove(f.Path)
		}
	}
}

//export Java_org_gioui_WaGoBridge_nativeDetach
func Java_org_gioui_WaGoBridge_nativeDetach(env *C.JNIEnv, cls C.jclass) { requests.detach() }

func OpenTarget(target string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := request(ctx, 3, target)
	return err
}

var foreground struct {
	sync.Mutex
	callback func()
}

func SetForeground(callback func()) {
	foreground.Lock()
	foreground.callback = callback
	foreground.Unlock()
}

//export Java_org_gioui_WaGoBridge_nativeForeground
func Java_org_gioui_WaGoBridge_nativeForeground(env *C.JNIEnv, cls C.jclass) {
	foreground.Lock()
	callback := foreground.callback
	foreground.Unlock()
	if callback != nil {
		callback()
	}
}
