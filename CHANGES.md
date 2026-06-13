# Fork Changes from Upstream (filebrowser/filebrowser)

Daftar perubahan custom di fork ini dibanding upstream.
Update terakhir: 2026-06-08

---

## 1. Fitur Chmod (Change File Permissions)

UI modal untuk ubah permission file/folder via checkbox atau input octal.

**File baru:**
- `http/chmod.go` — Backend handler `POST /api/chmod`
- `frontend/src/components/prompts/Chmod.vue` — Modal UI permission

**File dimodifikasi:**
- `http/http.go` — Register route `api.Handle("/api/chmod", ...)`
- `frontend/src/components/prompts/Prompts.vue` — Register komponen Chmod
- `frontend/src/api/files.ts` — Tambah fungsi `chmod(path, mode, recursive)`
- `frontend/src/views/files/FileListing.vue` — Tambah `headerButtons.chmod` + action di context menu
- `frontend/src/i18n/en.json` — Tambah translations: `buttons.chmod`, `buttons.apply`, `prompts.chmod*`

---

## 2. Fitur SystemUID/GID & Ownership

Setiap file/folder yang dibuat oleh user otomatis di-chown ke UID/GID yang dikonfigurasi.

**File baru:**
- `fileutils/ownership.go` — Struct `Ownership` + method `Chown()`

**File dimodifikasi:**
- `users/users.go` — Tambah field `SystemUID`, `SystemGID` + method `Ownership()`
- `http/resource.go` — Pakai `writeFileOwned()` dan `chownDirTree()` untuk chown setelah create
- `http/tus_handlers.go` — Chown setelah TUS upload selesai
- `fileutils/copy.go` — `CopyOwned()` variant
- `fileutils/dir.go` — `CopyDirOwned()` variant
- `fileutils/file.go` — `CopyFileOwned()` variant + `RealPath()` support `ScopedFs`

---

## 3. Fitur Compress/Archive (buat file archive di server)

Compress file/folder menjadi archive yang disimpan di server (terpisah dari download).

**File baru/dimodifikasi:**
- `http/archive.go` — Handler `POST /api/archive` dan `POST /api/extract`
- `http/http.go` — Register route archive & extract
- `frontend/src/components/prompts/Archive.vue` — Modal UI compress
- `frontend/src/views/files/FileListing.vue` — Tombol compress di context menu

---

## 4. Default File/Dir Permission

Ubah default permission dari upstream (`0640`/`0750`) ke `0644`/`0755`.

**File dimodifikasi:**
- `settings/settings.go` — `DefaultFileMode = 0644`, `DefaultDirMode = 0755`

---

## 5. Security: ScopedFs (merged from upstream v2.63.14)

Symlink escape CVE fix — replace `WithinScope()` checks dengan filesystem-level enforcement.

**File baru:**
- `files/scoped.go` — `ScopedFs` struct yang auto-block symlink escape di setiap operasi

**File dimodifikasi:**
- `files/file.go` — Hapus fungsi `WithinScope()`, update symlink handling di `readListing`
- `users/users.go` — Field `Fs` berubah type: `afero.Fs` → `*files.ScopedFs`
- `http/resource.go` — Hapus `WithinScope` guard di copy/move dan writeFile
- `http/tus_handlers.go` — Hapus `WithinScope` guard
- `http/raw.go` — Hapus `WithinScope` guard di `getFiles()`
- `http/share.go` — Hapus `WithinScope` guard di share creation
- `http/public.go` — Pakai `files.NewScopedFs()` bukan `afero.NewBasePathFs()`
- `http/chmod.go` — Pakai `d.user.Fs.Base()` untuk resolve path
- `http/archive.go` — Hapus `WithinScope` guard
- `fileutils/file.go` — `RealPath()` support interface `baser` untuk `ScopedFs`
- `files/file_test.go` — Rewrite tests ke pakai `ScopedFs`

**CVE yang di-fix:**
- GHSA-c2gv-wf5f-hjhh
- GHSA-239w-m3h6-ch8v

---

## Catatan

- Upstream repo: https://github.com/filebrowser/filebrowser
- Fork ini based on upstream v2.63.13, merged security fix dari v2.63.14
- Prioritas platform: Linux (VPS). Windows hanya untuk development.
