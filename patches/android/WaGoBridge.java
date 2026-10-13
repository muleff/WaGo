package org.gioui;

import android.Manifest;
import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.content.ClipData;
import android.content.pm.PackageManager;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.provider.OpenableColumns;
import android.webkit.MimeTypeMap;
import org.json.JSONArray;
import org.json.JSONObject;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.lang.ref.WeakReference;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public final class WaGoBridge {
 private static final int PICK = 5911, MIC = 5912;
 private static WeakReference<Activity> activity = new WeakReference<>(null);
 private static final Handler main = new Handler(Looper.getMainLooper());
 private static final ExecutorService worker = Executors.newSingleThreadExecutor();
 private static long pickID, micID;
 public static void attach(Activity a) { activity = new WeakReference<>(a); }
 public static void detach(Activity a) {
  if (activity.get() != a) return;
  activity.clear(); pickID = micID = 0; nativeDetach();
 }
 private static native void nativeResult(long id, byte[] payload);
 private static native void nativeDetach();
 private static native void nativeForeground();
 public static void resume(){nativeForeground();}
 private static void reply(long id, JSONArray files, String error) {
  try {
   JSONObject r = new JSONObject(); r.put("files", files); r.put("error", error);
   nativeResult(id, r.toString().getBytes(StandardCharsets.UTF_8));
  } catch (Exception e) { throw new IllegalStateException(e); }
 }
 private static void fail(long id, Exception e) { reply(id, null, e.getMessage() == null ? e.toString() : e.getMessage()); }
 public static void request(Context context, long id, int kind, String payload) {
  main.post(new Runnable() { @Override public void run() {
   Activity a = activity.get();
   if (a == null || a.isFinishing()) {reply(id, null, "Android activity is unavailable"); return;}
   try {
    if (kind == 0) {
     if (Build.VERSION.SDK_INT < 23 || a.checkSelfPermission(Manifest.permission.RECORD_AUDIO) == PackageManager.PERMISSION_GRANTED) {
      reply(id, null, ""); return;
     }
     if (micID != 0) {reply(id, null, "A microphone request is already pending");return;}
     micID = id; a.requestPermissions(new String[]{Manifest.permission.RECORD_AUDIO}, MIC);
    } else if (kind == 1) {
     if (pickID != 0) {reply(id, null, "A file picker is already open");return;}
     JSONObject options = new JSONObject(payload);
     Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT).addCategory(Intent.CATEGORY_OPENABLE).setType("*/*");
     intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
     intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, options.optBoolean("multiple"));
     JSONArray types = options.optJSONArray("mime");
     if (types != null && types.length() > 0) {
      String[] values = new String[types.length()]; for (int i = 0; i < values.length; i++) values[i] = types.getString(i);
      intent.putExtra(Intent.EXTRA_MIME_TYPES, values);
     }
     pickID = id;
     a.startActivityForResult(intent, PICK);
    } else if (kind == 2) {
     JSONObject options=new JSONObject(payload);
     File source=new File(options.getString("path")).getCanonicalFile();
     Uri uri=WaGoProvider.uriFor(a,source);
     String mime=options.optString("mime","application/octet-stream").split(";")[0].trim();
     Intent view=new Intent(Intent.ACTION_VIEW).setDataAndType(uri,mime);
     view.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
     view.setClipData(ClipData.newRawUri("media",uri));
     a.startActivity(view);reply(id,null,"");
    } else if(kind==3){
     Intent view;
     if("wago-settings:".equals(payload)){view=new Intent(android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS).setData(Uri.parse("package:"+a.getPackageName()));}
     else {
      Uri target=Uri.parse(payload);String scheme=target.getScheme();
      if(!"http".equals(scheme) && !"https".equals(scheme) && !"mailto".equals(scheme))throw new IllegalArgumentException("Unsupported link");
      view=new Intent(Intent.ACTION_VIEW,target);
     }
     a.startActivity(view);reply(id,null,"");
    } else reply(id, null, "Unknown Android request");
   } catch (Exception e) {if(pickID==id)pickID=0;if(micID==id)micID=0;fail(id,e);}
  } });
 }
 public static boolean permissions(int code, int[] grants) {
  if (code != MIC) return false;
  long id = micID; micID = 0;
  if (id != 0) reply(id, null, grants.length > 0 && grants[0] == PackageManager.PERMISSION_GRANTED ? "" : "permission");
  return true;
 }
 public static boolean picked(Activity a, int code, int status, Intent data) {
  if (code != PICK) return false;
  long id = pickID; pickID = 0;
  if (id == 0) return true;
  if (status != Activity.RESULT_OK || data == null) {reply(id, new JSONArray(), "");return true;}
  ArrayList<Uri> uris = new ArrayList<>();
  if (data.getClipData() != null) for (int i=0; i<data.getClipData().getItemCount(); i++) uris.add(data.getClipData().getItemAt(i).getUri());
  else if (data.getData() != null) uris.add(data.getData());
  Context app = a.getApplicationContext();
  worker.execute(new Runnable() { public void run() { copyFiles(app,id,uris); } });
  return true;
 }
 private static void copyFiles(Context app,long id,ArrayList<Uri> uris) {
  ArrayList<File> created = new ArrayList<>();
  try {
   if (uris.size() > 100) throw new java.io.IOException("Select at most 100 files");
   JSONArray files = new JSONArray();
   File root = new File(app.getCacheDir(), "wago-picked");
   if (!root.exists() && !root.mkdirs()) throw new java.io.IOException("Could not create attachment cache");
   for (Uri uri : uris) {
    String mime = app.getContentResolver().getType(uri), name = "attachment";
    try (Cursor c = app.getContentResolver().query(uri, new String[]{OpenableColumns.DISPLAY_NAME},null,null,null)) {
     if(c!=null && c.moveToFirst() && !c.isNull(0)) name = c.getString(0);
    }
    name = new File(name.replace('\\','/')).getName().replaceAll("[\\p{Cntrl}]", "_");
    if(name.isEmpty() || name.equals(".") || name.equals("..")) name="attachment";

    String ext = mime == null ? null : MimeTypeMap.getSingleton().getExtensionFromMimeType(mime);

    File dest=File.createTempFile("pick-",ext==null?".bin":"."+ext,root);created.add(dest);
    try (InputStream in = app.getContentResolver().openInputStream(uri); FileOutputStream out = new FileOutputStream(dest)) {
     if(in==null)throw new java.io.IOException("Document could not be opened");
     byte[] buffer=new byte[65536]; int n; while((n=in.read(buffer))!=-1)out.write(buffer,0,n);
    }
    JSONObject file=new JSONObject();file.put("Path",dest.getAbsolutePath());file.put("Name",name);file.put("MIME",mime);files.put(file);
   }
   reply(id,files,"");
  } catch (Exception e) {for(File f:created){f.delete();}fail(id,e);}
 }
}
