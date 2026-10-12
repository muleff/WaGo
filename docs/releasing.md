# Build otomatis dan rilis

Workflow `.github/workflows/build.yml` berjalan pada setiap push branch, pull
request, push tag `v*`, dan melalui **Actions > Build and release > Run workflow**.
Pilihan manual tersedia setelah workflow masuk ke default branch.

Versi Go mengikuti `go.mod`. Semua target menerapkan patch go-text
(`patches/apply.sh`), lalu menjalankan pemeriksaan format, `go mod verify`,
`go vet ./...`, `go test ./...`, dan `go build ./...` sebelum membangun aplikasi:

| File | Isi |
| --- | --- |
| `OpenWA-Setup.exe` | Installer Windows (Inno Setup, `installer/wazzap.iss`) |
| `OpenWA-windows-amd64.exe` | Aplikasi Windows portable, tanpa jendela console; juga file yang diunduh oleh tombol update |
| `OpenWA-windows-portable.zip` | Aplikasi Windows plus `ffmpeg.exe` untuk perekaman voice note tanpa instalasi FFmpeg terpisah |
| `OpenWA-linux-amd64` | Executable Linux |
| `OpenWA-android.apk` | APK Android untuk ARMv7 dan ARM64, minimum Android 5.0 (API 21), aplikasi Gio dengan backend WhatsApp |
| `SHA256SUMS`, `SHA256SUMS.sig` | Checksum semua file di atas, dan tanda tangan ed25519-nya (hanya di release) |

Unduh hasil dari bagian **Artifacts** pada run yang berhasil. Artefak disimpan
selama 14 hari; aset yang sudah dilampirkan ke GitHub Release tetap tersimpan.

Linux dibangun di Ubuntu 22.04 dengan CGO serta dukungan X11/Wayland. Paket ini
bukan binary statis atau AppImage; komputer tujuan memerlukan lingkungan desktop
dan library runtime Gio (X11/Wayland, xkbcommon, EGL/GLES). Lihat
[dependensi Linux Gio](https://gioui.org/doc/install/linux). macOS, Linux ARM64, dan
code signing Windows belum disediakan oleh workflow ini.

## APK Android (Gio)

Job `android` menginstal Android SDK 35, Build Tools 35, NDK 27, Java 17, dan
`gioui.org/cmd/gogio@v0.10.3`. Perintah `gogio` mengemas `./cmd/android`
sebagai `com.muleff.wago` dengan dua library native (`armeabi-v7a` dan
`arm64-v8a`); APK diperiksa dengan `unzip`, `apksigner` dan `aapt`.
Hasil setiap build tersedia dari tab Actions > Artifacts selama 14 hari.
Build tag `vMAJOR.MINOR.PATCH` menyertakan APK dalam draft release, checksums
dan tanda tangan checksums seperti executable lain.

APK ditandatangani otomatis menggunakan kunci sementara untuk pengujian.
Untuk memungkinkan pembaruan tanpa menghapus aplikasi, simpan kedua secret
repository berikut di **Settings > Secrets and variables > Actions > Secrets**:

- `ANDROID_KEYSTORE_BASE64`: seluruh file keystore Android (JKS/PKCS12) sebagai base64;
  keystore sebaiknya berisi satu signing key.
- `ANDROID_KEYSTORE_PASSWORD`: kata sandi keystore yang sama. Jangan commit
  file keystore atau nilai secret ke repository.

Jika hanya salah satu secret disetel, build Android akan gagal daripada
menandatangani dengan identitas yang salah. Jika keduanya tidak disetel,
APK debug tetap dapat dipasang, tetapi tidak selalu bisa memperbarui APK
sebelumnya karena sertifikat tanda tangannya berubah.

Android menggunakan UI seluler Gio dan backend `wa` yang nyata, bukan demo.
Namun sejumlah integrasi desktop belum ada di Android, termasuk perekaman
voice note (saat ini masih bergantung pada FFmpeg di desktop). Build APK yang
lulus belum sama dengan pengujian perangkat Android nyata.

## Installer Windows

Workflow Windows mengambil FFmpeg Essentials dari Gyan, memverifikasi SHA-256
arsip resmi, kemudian menyertakan `ffmpeg.exe` dan informasi lisensinya bersama
OpenWA. Installer dan ZIP portable dapat merekam voice note tanpa instalasi
FFmpeg di PATH. File `.exe` tunggal masih membutuhkan FFmpeg pada PC atau di
folder yang sama. FFmpeg didistribusikan dengan lisensinya; sumber proyek:
https://ffmpeg.org/download.html dan https://www.gyan.dev/ffmpeg/builds/.

Installer memasang aplikasi untuk user yang sedang login saja, ke
`%LocalAppData%\Programs\WazzapClients`, tanpa meminta hak admin. Installer membuat
shortcut di Start Menu (opsional di Desktop) dan mendaftarkan uninstaller di
**Settings > Apps**. Uninstaller menutup aplikasi, menghapus entri registry yang
dibuat aplikasi sendiri (notifikasi dan start at login), lalu menanyakan apakah
chat dan sesi WhatsApp (`%AppData%\WazzapClients`) ikut dihapus. Defaultnya tidak.

Untuk membangunnya secara lokal, pasang [Inno Setup 6](https://jrsoftware.org/isinfo.php),
build exe ke `bin/dist/OpenWA-windows-amd64.exe`, lalu jalankan
`iscc /DAppVersion=0.10.0 installer\wazzap.iss`. Salin pula `ffmpeg.exe`
dan `ffmpeg-LICENSE.txt` ke `bin/dist` sebelum kompilasi installer.

## Update dari dalam aplikasi

**Settings > Help** menampilkan versi aplikasi dan tombol **Check for updates**.
Aplikasi tidak menghubungi GitHub sampai tombol itu ditekan. Kalau ada versi yang
lebih baru, tombol yang sama mengunduh exe untuk sistem ini, mencocokkannya dengan
`SHA256SUMS`, memeriksa tanda tangan `SHA256SUMS.sig` dengan kunci publik di
`internal/update`, menukar exe yang sedang berjalan, lalu me-restart aplikasi.

- Hanya build dari tag (`-X main.version=v1.2.3`) yang bisa update. Build lain,
  dan prerelease seperti `v1.2.3-rc1`, adalah development build.
- Yang dicek adalah *latest release* GitHub, jadi draft dan prerelease tidak
  pernah ditawarkan.
- Update butuh folder exe yang bisa ditulis. Installer memasang ke folder seperti
  itu; untuk versi portable, letakkan exe di folder milik user.

### Kunci tanda tangan

Job release menandatangani `SHA256SUMS` dengan `cmd/signrelease` memakai secret
`RELEASE_SIGNING_KEY` (seed ed25519 dalam base64). Tanpa secret itu job release
gagal. Kunci publiknya ada di `publicKey` (`internal/update/update.go`).

Simpan cadangan kunci privat di tempat aman. Kalau kunci hilang atau bocor, buat
kunci baru (`go run ./cmd/signrelease -keygen <file>`, yang mencetak kunci publik
barunya), ganti `publicKey` dan secret-nya. Versi lama aplikasi tidak akan mau
memasang rilis yang ditandatangani kunci baru, jadi penggunanya harus mengunduh
satu rilis itu secara manual.

## Membuat rilis

1. Commit dan push source beserta workflow ke repository GitHub.
2. Pada commit yang akan dirilis, buat dan push tag versi baru, misalnya:

   ```sh
   git tag -a v0.1.0 -m "WazzapClients v0.1.0"
   git push origin v0.1.0
   ```

3. Setelah build Windows, Linux, dan Android berhasil, workflow membuat **draft release** dengan semua
   file di atas dan release notes otomatis.
4. Buka **Releases**, periksa draft, sunting catatan rilis (dan tandai prerelease
   bila diperlukan), lalu pilih **Publish release**. Baru setelah itu aplikasi
   menawarkannya sebagai update.

Hanya job release yang memiliki izin `contents: write`. Kebijakan
repository/organisasi harus mengizinkan GitHub Actions membuat release.

Menjalankan ulang run tag memperbarui aset selama release masih draft. Aset
release yang telah dipublikasikan tidak ditimpa; gunakan tag versi baru.
Run manual dan push branch hanya menghasilkan artefak, tanpa membuat release.

Untuk memeriksa unduhan di Linux, simpan file dan `SHA256SUMS` dalam satu
direktori, lalu jalankan `sha256sum --check --ignore-missing SHA256SUMS`. Di
PowerShell, gunakan `Get-FileHash .\OpenWA-Setup.exe -Algorithm SHA256`
dan cocokkan hasilnya dengan entri di `SHA256SUMS`.
