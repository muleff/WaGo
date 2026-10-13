#include <jni.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

char *wago_bytes(JNIEnv *env, jbyteArray bytes, int *len) {
 if (!bytes) return NULL;
 *len = (*env)->GetArrayLength(env, bytes);
 char *out = malloc(*len + 1);
 if (!out) return NULL;
 (*env)->GetByteArrayRegion(env, bytes, 0, *len, (jbyte *)out);
 out[*len] = 0;
 return out;
}

int wago_request(uintptr_t vmPtr, uintptr_t ctxPtr, long long id, int kind, const char *payload) {
 JavaVM *vm = (JavaVM *)vmPtr;
 if (!vm || !ctxPtr) return -1;
 JNIEnv *env = NULL;
 int detach = 0, status = -1;
 jint rc = (*vm)->GetEnv(vm, (void **)&env, JNI_VERSION_1_6);
 if (rc == JNI_EDETACHED) {
  if ((*vm)->AttachCurrentThread(vm, &env, NULL) != JNI_OK) return -1;
  detach = 1;
 } else if (rc != JNI_OK) return -1;
 (*env)->PushLocalFrame(env, 16);
 jobject ctx = (jobject)ctxPtr;
 jclass context = (*env)->GetObjectClass(env, ctx);
 jmethodID getLoader = (*env)->GetMethodID(env, context, "getClassLoader", "()Ljava/lang/ClassLoader;");
 jobject loader = (*env)->CallObjectMethod(env, ctx, getLoader);
 jclass loaderClass = (*env)->GetObjectClass(env, loader);
 jmethodID load = (*env)->GetMethodID(env, loaderClass, "loadClass", "(Ljava/lang/String;)Ljava/lang/Class;");
 jstring name = (*env)->NewStringUTF(env, "org.gioui.WaGoBridge");
 jclass bridge = (jclass)(*env)->CallObjectMethod(env, loader, load, name);
 if (!(*env)->ExceptionCheck(env) && bridge) {
  jmethodID request = (*env)->GetStaticMethodID(env, bridge, "request", "(Landroid/content/Context;JILjava/lang/String;)V");
  if (request) {
   jbyteArray bytes = (*env)->NewByteArray(env, strlen(payload));
   (*env)->SetByteArrayRegion(env, bytes, 0, strlen(payload), (const jbyte *)payload);
   jclass stringClass = (*env)->FindClass(env, "java/lang/String");
   jmethodID constructor = (*env)->GetMethodID(env, stringClass, "<init>", "([BLjava/lang/String;)V");
   jstring charset = (*env)->NewStringUTF(env, "UTF-8");
   jstring data = (jstring)(*env)->NewObject(env, stringClass, constructor, bytes, charset);
   (*env)->CallStaticVoidMethod(env, bridge, request, ctx, (jlong)id, (jint)kind, data);
   if (!(*env)->ExceptionCheck(env)) status = 0;
  }
 }
 if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
 (*env)->PopLocalFrame(env, NULL);
 if (detach) (*vm)->DetachCurrentThread(vm);
 return status;
}
