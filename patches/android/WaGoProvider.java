package org.gioui;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.content.Context;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.provider.OpenableColumns;
import android.webkit.MimeTypeMap;
import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.util.List;

// URI permission grants expose only the requested private file; reuse the
// existing media cache rather than retaining another full copy on every open.
public final class WaGoProvider extends ContentProvider {
 public boolean onCreate(){return true;}
 public static Uri uriFor(Context context,File source)throws IOException{
  source=source.getCanonicalFile();if(!source.isFile())throw new FileNotFoundException();
  for(String bucket:new String[]{"files","cache"}){
   File root=(bucket.equals("files")?context.getFilesDir():context.getCacheDir()).getCanonicalFile();
   String prefix=root.getPath()+File.separator;
   if(source.getPath().startsWith(prefix))return new Uri.Builder().scheme("content").authority(context.getPackageName()+".media")
    .appendPath(bucket).appendPath(source.getPath().substring(prefix.length())).build();
  }
  throw new FileNotFoundException("File is outside app storage");
 }
 private File file(Uri uri)throws FileNotFoundException{
  try {
   List<String> parts=uri.getPathSegments();if(parts.size()!=2)throw new FileNotFoundException();
   String bucket=parts.get(0);if(!bucket.equals("files")&&!bucket.equals("cache"))throw new FileNotFoundException();
   File root=(bucket.equals("files")?getContext().getFilesDir():getContext().getCacheDir()).getCanonicalFile();
   File f=new File(root,parts.get(1)).getCanonicalFile();
   if(!f.getPath().startsWith(root.getPath()+File.separator)||!f.isFile())throw new FileNotFoundException();
   return f;
  }catch(IOException|NullPointerException e){throw new FileNotFoundException();}
 }
 public ParcelFileDescriptor openFile(Uri uri,String mode)throws FileNotFoundException{
  if(!"r".equals(mode))throw new FileNotFoundException("Read only");
  return ParcelFileDescriptor.open(file(uri),ParcelFileDescriptor.MODE_READ_ONLY);
 }
 public String getType(Uri uri){
  String name=uri.getLastPathSegment();int dot=name==null?-1:name.lastIndexOf('.');
  String mime=dot<0?null:MimeTypeMap.getSingleton().getMimeTypeFromExtension(name.substring(dot+1).toLowerCase(java.util.Locale.ROOT));
  return mime==null?"application/octet-stream":mime;
 }
 public Cursor query(Uri uri,String[] projection,String selection,String[] args,String order){
  try {
   File f=file(uri);if(projection==null)projection=new String[]{OpenableColumns.DISPLAY_NAME,OpenableColumns.SIZE};
   MatrixCursor c=new MatrixCursor(projection);Object[] row=new Object[projection.length];
   for(int i=0;i<projection.length;i++){
    if(OpenableColumns.DISPLAY_NAME.equals(projection[i]))row[i]=f.getName();
    if(OpenableColumns.SIZE.equals(projection[i]))row[i]=f.length();
   }
   c.addRow(row);return c;
  }catch(FileNotFoundException e){return null;}
 }
 public Uri insert(Uri uri,ContentValues v){throw new UnsupportedOperationException();}
 public int update(Uri uri,ContentValues v,String s,String[] args){throw new UnsupportedOperationException();}
 public int delete(Uri uri,String s,String[] args){throw new UnsupportedOperationException();}
}
