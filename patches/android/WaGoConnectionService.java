package org.gioui;

import android.app.Service;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ServiceInfo;
import android.os.Build;
import android.os.IBinder;
import android.os.PowerManager;

// The visible foreground notification keeps Android from treating linking or
// an active WhatsApp socket as disposable work when the Activity is covered.
public final class WaGoConnectionService extends Service {
 private PowerManager.WakeLock linkingWake;
 public static void start(Context context) {
  Intent intent=new Intent(context,WaGoConnectionService.class);
  if(Build.VERSION.SDK_INT>=26)context.startForegroundService(intent);
  else context.startService(intent);
 }
 @Override public void onCreate(){
  super.onCreate();
  String channel="wago_connection";
  if(Build.VERSION.SDK_INT>=26){
   NotificationManager manager=(NotificationManager)getSystemService(NOTIFICATION_SERVICE);
   manager.createNotificationChannel(new NotificationChannel(channel,"Conexión de WaGo",NotificationManager.IMPORTANCE_LOW));
  }
  Intent open=new Intent(this,GioActivity.class).addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP);
  int flags=PendingIntent.FLAG_UPDATE_CURRENT;
  if(Build.VERSION.SDK_INT>=23)flags|=PendingIntent.FLAG_IMMUTABLE;
  PendingIntent pending=PendingIntent.getActivity(this,0,open,flags);
  Notification.Builder builder=Build.VERSION.SDK_INT>=26?new Notification.Builder(this,channel):new Notification.Builder(this);
  Notification notification=builder.setSmallIcon(android.R.drawable.stat_notify_sync).setContentTitle("WaGo")
   .setContentText("Manteniendo la conexión en segundo plano").setOngoing(true).setContentIntent(pending).build();
  if(Build.VERSION.SDK_INT>=34)startForeground(5910,notification,ServiceInfo.FOREGROUND_SERVICE_TYPE_REMOTE_MESSAGING);
  else if(Build.VERSION.SDK_INT>=29)startForeground(5910,notification,ServiceInfo.FOREGROUND_SERVICE_TYPE_DATA_SYNC);
  else startForeground(5910,notification);
  // A bounded wake lock protects the short phone-code linking round-trip.
  PowerManager power=(PowerManager)getSystemService(POWER_SERVICE);
  linkingWake=power.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK,"WaGo:linking");
  linkingWake.acquire(10*60*1000L);
  Gio.init(getApplicationContext());
 }
 @Override public int onStartCommand(Intent intent,int flags,int startId){return START_STICKY;}
 @Override public IBinder onBind(Intent intent){return null;}
 @Override public void onTimeout(int startId,int type){stopSelf();}
 @Override public void onDestroy(){if(linkingWake!=null && linkingWake.isHeld())linkingWake.release();super.onDestroy();}
}
