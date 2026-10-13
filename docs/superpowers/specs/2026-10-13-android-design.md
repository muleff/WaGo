# WaGo Android: funcionalidad y experiencia móvil

## Objetivo y alcance aprobado

Hacer que el cliente Android existente permita conversar, seleccionar y enviar
media, grabar y reproducir notas de voz y navegar cómodamente en español. Mantener
Go/Gio, hypermeow, las sesiones existentes y el objetivo de bajo consumo de memoria.
La referencia es la familiaridad de WhatsApp; WaGo conserva su identidad propia.

## Evidencia inicial

- `internal/filepick/filepick_other.go` incluye Android y sólo ofrece herramientas
  de escritorio: sin ellas devuelve `ErrUnsupported`.
- `cmd/android/main.go` añade permisos al manifiesto, pero
  `internal/voicerec/micpermission_other.go` devuelve nil sin pedir permiso Android.
- `internal/ui/mobile.go` dibuja seis destinos sólo con iconos y separa canales
  de estados. El botón Atrás del sistema se consume sin ejecutar navegación.
- El panel de información reutiliza dimensiones de escritorio y se superpone a
  una conversación dibujada detrás. La captura muestra dos acciones de regreso.
- `UI.label` usa `displayText` sin localizar el texto. El catálogo español existe,
  pero varias etiquetas de información se pasan como cadenas inglesas.
- La lista tipográfica pone la fuente emoji antes de Roboto y Go; su cobertura de
  dígitos es una hipótesis para el espaciado de números, pendiente de reproducción.
- Hay captura y reproducción Ogg/Opus Android existentes. Se deben auditar y probar,
  no sustituir a ciegas. El reproductor actual no soporta imágenes de video.

## Decisión de arquitectura

Adaptar el proyecto existente mediante una capa Android aislada para permisos,
selección de documentos y apertura de media con APIs del sistema. Es preferible a
reemplazar la aplicación por una WebView o reescribirla: preserva sesiones y backend
y concentra los cambios en las integraciones que faltan. Leer las APIs de Gio y
hypermeow de las versiones fijadas antes de implementarlas.

La capa Android entrega resultados asíncronos al código Go. La UI recibe cambios
en su goroutine mediante el mecanismo de resultados e invalidación existente.
No bloquear el hilo de actividad ni modificar widgets desde callbacks nativos.

## Archivos y permisos

Usar Storage Access Framework para imágenes, videos, audio y documentos. Resolver
`content://` con ContentResolver, copiar por streaming a caché privada y entregar
una ruta utilizable por el backend actual. Conservar nombre y MIME cuando sea
posible; cerrar descriptores y limpiar temporales al cancelar, fallar o terminar
el envío. No intentar interpretar una URI como ruta del filesystem.

Solicitar RECORD_AUDIO al iniciar una grabación, esperar el resultado y sólo luego
iniciar captura. Si se deniega, explicar el siguiente paso en español sin bucles
de solicitudes. No exigir permisos de almacenamiento global para elegir archivos.
Solicitar cámara únicamente cuando exista una acción de cámara implementada.

## Audio y conexión

Validar captura PCM y codificación Ogg/Opus compatible con mensajes PTT. Revisar
orden de cierre, drenado de frames y cancelación para evitar truncamiento y fugas.
Probar pausa, reanudación, búsqueda y final de reproducción; el estado debe seguir
los samples efectivamente consumidos, sin bloquear comandos detrás de una cola llena.
Usar reproductor Android para formatos y videos que el backend actual no soporte.

Auditar el enlace y las operaciones de envío: iniciar sólo con sesión conectada,
representar desconexión/reconexión en español y conservar el borrador. Un fallo de
red no debe aparecer como envío exitoso. No enviar mensajes de prueba a contactos
reales sin que el usuario indique un destinatario autorizado.

## Diseño móvil

- Navegación inferior con cuatro destinos etiquetados: Chats, Novedades,
  Comunidades y Llamadas. Ajustes y cuenta accesibles desde el menú superior.
- Novedades reúne estados y canales en una superficie desplazable; conserva los
  flujos existentes de visualización, publicación y apertura de canales.
- Chat, información del contacto, búsquedas y visor ocupan el ancho completo.
  Cada pantalla de detalle tiene una sola acción visible de regreso.
- Atrás cierra primero diálogo/visor/panel y luego vuelve a la lista; en raíz
  respeta el comportamiento del sistema. El teclado no tapa el compositor.
- Objetivos táctiles de 48 dp, texto principal de aproximadamente 16 sp,
  safe areas, tema claro/oscuro y tamaños de letra ajustables.
- Los números usan una fuente de texto, sin separación artificial entre dígitos;
  no se traducen nombres, números, enlaces ni contenido de mensajes.
- Localizar etiquetas, acciones, estados vacíos y errores propios de la aplicación.
  Usar claves y formatos explícitos para textos dinámicos, sin traducir datos del usuario.

## Verificación y aceptación

1. Pruebas de regresión para navegación Atrás, información a ancho completo,
   Novedades y traducciones de las cadenas observadas en la captura.
2. Renderizar tamaños 320 y 390 dp, landscape y viewport con teclado; comprobar
   dígitos, cabecera única y ausencia de recortes. Un test que sólo verifica el
   tamaño exterior del frame no basta.
3. Pruebas de temporales/cancelación del selector, finalización de audio y manejo
   de desconexión; `gofmt`, `go vet`, tests y builds exigidos por AGENTS.md.
4. Compilar APK ARM y ARM64, verificar manifiesto, firma y contenido. En emulador
   comprobar permisos concedidos/denegados, selector real y ciclo de actividad.
5. Al terminar, inspeccionar workflows de GitHub, identificar commit y estado de
   los jobs y reportar fallos concretos. Nunca incluir el token en archivos o logs.

La validación de recepción WhatsApp real requiere sesión y destinatario autorizado;
un build exitoso o pruebas mock no demuestran recepción. Informar por separado las
pruebas ejecutadas, las limitaciones de entorno y lo que queda por validar.

## Entorno actual

El repositorio está clonado localmente. El shell actual no tiene `go`; se debe
habilitar un toolchain compatible antes de afirmar resultados de compilación.
No hay APK generado ni pruebas de dispositivo ejecutadas en esta fase.
