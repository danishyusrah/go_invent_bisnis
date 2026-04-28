# Aplikasi Manajemen Inventaris - Jack Sound Audio

Aplikasi web internal yang dirancang untuk pelacakan stok, harga, dan lokasi fisik sparepart dan komponen elektronik secara efisien. Dibangun dengan Go (Gin) dan MySQL, aplikasi ini menyediakan antarmuka yang bersih, cepat, dan fungsional untuk mengelola inventaris harian.


---

## ✨ Fitur Utama

Aplikasi ini dilengkapi dengan serangkaian fitur kompleks yang siap pakai untuk manajemen inventaris profesional:

### 1. Dashboard Analitis
- **Metrik "At-a-Glance"**: Menampilkan 4 kartu statistik utama secara real-time:
  - **Total SKU Aktif**: Jumlah total jenis barang yang terdaftar.
  - **Stok Habis (Out of Stock)**: Jumlah barang dengan stok 0.
  - **Peringatan Stok Rendah**: Jumlah barang yang stoknya di bawah batas minimum.
  - **Total Nilai Inventaris**: Estimasi total nilai aset berdasarkan harga modal (harga beli).
- **Tabel Stok Kritis**: Daftar prioritas barang-barang yang stoknya menipis atau habis, memungkinkan tindakan cepat untuk re-order.

### 2. Manajemen Inventaris (CRUD Penuh)
- **Penambahan Produk Baru (Create)**: Form khusus untuk menambahkan barang baru ke database dengan detail lengkap (Part Number, Deskripsi, Kuantitas, Harga Modal, dll).
- **Daftar Seluruh Inventaris (Read)**: Halaman `/inventory` yang menampilkan **semua produk** yang terdaftar, tidak hanya yang stoknya rendah.
- **Edit Produk (Update)**: Kemampuan untuk mengedit detail setiap produk melalui form yang sudah terisi data sebelumnya.
- **Hapus Produk (Delete)**: Fungsi untuk menghapus produk dari database dengan dialog konfirmasi untuk mencegah kesalahan.

### 3. Fungsionalitas Pendukung
- **Paginasi**: Semua halaman daftar (list) dilengkapi dengan sistem paginasi untuk menangani data dalam jumlah besar tanpa membebani browser atau server.
- **Notifikasi User (Flash Messages)**: Pesan umpan balik (misalnya "Produk berhasil ditambahkan" atau "Gagal menghapus produk") muncul setelah setiap operasi CRUD untuk memberitahu status aksi kepada pengguna.
- **Konfigurasi Terpusat**: Seluruh konfigurasi penting (koneksi database, nama aplikasi, port) dikelola melalui file `.env`, sehingga tidak ada *hardcoding* dan mudah disesuaikan untuk lingkungan development atau production.
- **Antarmuka Bersih & Responsif**: Didesain menggunakan Tailwind CSS, memastikan tampilan yang optimal di berbagai ukuran layar, dari desktop hingga mobile.

---

## 🛠️ Teknologi yang Digunakan

- **Backend**: Go (v1.21+)
- **Web Framework**: Gin Gonic
- **Database**: MySQL (v8.0+)
- **Frontend**: HTML5, Tailwind CSS (via CDN)
- **Manajemen Konfigurasi**: `godotenv` untuk file `.env`
- **Session Management**: `gin-contrib/sessions` untuk Flash Messages

untuk ui ux nya saya ingin seperti ini
# Aplikasi Manajemen Inventaris - Jack Sound Audio

Aplikasi web internal yang dirancang untuk pelacakan stok, harga, dan lokasi fisik sparepart dan komponen elektronik secara efisien. Dibangun dengan Go (Gin) dan MySQL, aplikasi ini menyediakan antarmuka yang bersih, cepat, dan fungsional untuk mengelola inventaris harian.

![Dashboard Screenshot](https://i.ibb.co/L5w2r2d/jacksound-dashboard.png)
*(Gantilah URL di atas dengan URL screenshot aplikasi Anda setelah berjalan)*

---

## ✨ Fitur Utama

Aplikasi ini dilengkapi dengan serangkaian fitur kompleks yang siap pakai untuk manajemen inventaris profesional:

### 1. Dashboard Analitis
- **Metrik "At-a-Glance"**: Menampilkan 4 kartu statistik utama secara real-time:
  - **Total SKU Aktif**: Jumlah total jenis barang yang terdaftar.
  - **Stok Habis (Out of Stock)**: Jumlah barang dengan stok 0.
  - **Peringatan Stok Rendah**: Jumlah barang yang stoknya di bawah batas minimum.
  - **Total Nilai Inventaris**: Estimasi total nilai aset berdasarkan harga modal (harga beli).
- **Tabel Stok Kritis**: Daftar prioritas barang-barang yang stoknya menipis atau habis, memungkinkan tindakan cepat untuk re-order.

### 2. Manajemen Inventaris (CRUD Penuh)
- **Penambahan Produk Baru (Create)**: Form khusus untuk menambahkan barang baru ke database dengan detail lengkap (Part Number, Deskripsi, Kuantitas, Harga Modal, dll).
- **Daftar Seluruh Inventaris (Read)**: Halaman `/inventory` yang menampilkan **semua produk** yang terdaftar, tidak hanya yang stoknya rendah.
- **Edit Produk (Update)**: Kemampuan untuk mengedit detail setiap produk melalui form yang sudah terisi data sebelumnya.
- **Hapus Produk (Delete)**: Fungsi untuk menghapus produk dari database dengan dialog konfirmasi untuk mencegah kesalahan.

### 3. Fungsionalitas Pendukung
- **Paginasi**: Semua halaman daftar (list) dilengkapi dengan sistem paginasi untuk menangani data dalam jumlah besar tanpa membebani browser atau server.
- **Notifikasi User (Flash Messages)**: Pesan umpan balik (misalnya "Produk berhasil ditambahkan" atau "Gagal menghapus produk") muncul setelah setiap operasi CRUD untuk memberitahu status aksi kepada pengguna.
- **Konfigurasi Terpusat**: Seluruh konfigurasi penting (koneksi database, nama aplikasi, port) dikelola melalui file `.env`, sehingga tidak ada *hardcoding* dan mudah disesuaikan untuk lingkungan development atau production.
- **Antarmuka Bersih & Responsif**: Didesain menggunakan Tailwind CSS, memastikan tampilan yang optimal di berbagai ukuran layar, dari desktop hingga mobile.

---

## 🛠️ Teknologi yang Digunakan

- **Backend**: Go (v1.21+)
- **Web Framework**: Gin Gonic
- **Database**: MySQL (v8.0+)
- **Frontend**: HTML5, Tailwind CSS (via CDN)
- **Manajemen Konfigurasi**: `godotenv` untuk file `.env`
- **Session Management**: `gin-contrib/sessions` untuk Flash Messages

---

## 🚀 Instalasi & Konfigurasi

Ikuti langkah-langkah berikut untuk menjalankan aplikasi ini di lingkungan lokal Anda.

### 1. Prasyarat
- [Go](https://go.dev/doc/install) versi 1.21 atau lebih baru.
- [MySQL](https://dev.mysql.com/downloads/installer/) versi 8.0 atau lebih baru.
- [Git](https://git-scm.com/downloads) untuk kloning repositori.

### 2. Langkah-langkah Instalasi

1.  **Kloning Repositori**
    ```bash
    git clone <URL_REPOSITORI_ANDA>
    cd inventory-app
    ```

2.  **Konfigurasi Lingkungan**
    Buat file bernama `.env` di root folder proyek dan isi dengan konfigurasi berikut. Ganti nilai sesuai dengan pengaturan lokal Anda.

    ```env
    # Database Configuration
    DB_USER="root"
    DB_PASSWORD="your_mysql_password" # <-- GANTI INI
    DB_HOST="127.0.0.1"
    DB_PORT="3306"
    DB_NAME="inventory_db"

    # Application Configuration
    APP_PORT="8080"
    APP_NAME="Jack Sound Audio"
    APP_DESCRIPTION="Inventory Control"
    ```

3.  **Setup Database**
    - Buka MySQL client Anda (misalnya DBeaver, MySQL Workbench, atau terminal).
    - Buat database baru sesuai dengan `DB_NAME` di file `.env`.
      ```sql
      CREATE DATABASE IF NOT EXISTS inventory_db;
      ```
    - Jalankan skrip SQL di bawah ini untuk membuat tabel `products` dan mengisi data awal.
      - **Skema Tabel**: `database/products_schema.sql`
      - **Data Sampel**: `database/dummy_data.sql`

4.  **Instalasi Dependensi**
    Buka terminal di root folder proyek dan jalankan perintah berikut untuk mengunduh semua package Go yang dibutuhkan.
    ```bash
    go mod tidy
    ```

### 3. Menjalankan Aplikasi

Setelah semua langkah di atas selesai, jalankan aplikasi dengan perintah:

```bash
go run main.go