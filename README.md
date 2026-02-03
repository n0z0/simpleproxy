# Simple SOCKS5 Proxy Server

Server proxy SOCKS5 sederhana yang ditulis dalam Go.

## Fitur

- ✅ Implementasi protokol SOCKS5 dasar
- ✅ Mendukung koneksi TCP
- ✅ Mendukung alamat IPv4, IPv6, dan domain name
- ✅ Tanpa autentikasi (no-auth mode)
- ✅ Relay data dua arah antara client dan target

## Cara Menggunakan

### Menjalankan Server

```bash
# Jalankan di port default (1080)
go run main.go

# Atau jalankan di port custom
go run main.go 8080
```

### Build Binary

```bash
# Build untuk Windows
go build -o socks5-proxy.exe main.go

# Build untuk Linux
GOOS=linux GOARCH=amd64 go build -o socks5-proxy main.go

# Build untuk macOS
GOOS=darwin GOARCH=amd64 go build -o socks5-proxy main.go
```

### Testing dengan cURL

```bash
# Test koneksi melalui proxy
curl -x socks5://localhost:1080 https://www.google.com

# Test dengan port custom
curl -x socks5://localhost:8080 https://www.google.com
```

### Konfigurasi Browser

Untuk menggunakan proxy di browser:

1. **Firefox**:
   - Settings → Network Settings → Manual proxy configuration
   - SOCKS Host: `localhost`
   - Port: `1080`
   - Pilih SOCKS v5

2. **Chrome/Edge**:
   - Gunakan extension seperti "Proxy SwitchyOmega"
   - Atau jalankan dengan flag: `--proxy-server="socks5://localhost:1080"`

## Struktur Kode

- `handshake()` - Menangani SOCKS5 handshake awal
- `handleRequest()` - Memproses request koneksi dan connect ke target
- `sendReply()` - Mengirim response ke client
- `relay()` - Meneruskan data antara client dan target server

## Limitasi

- Hanya mendukung command CONNECT (tidak mendukung BIND atau UDP ASSOCIATE)
- Tidak ada autentikasi (semua koneksi diterima)
- Tidak ada logging detail atau metrics
- Tidak ada rate limiting atau access control

## Pengembangan Lebih Lanjut

Untuk production use, pertimbangkan untuk menambahkan:
- Autentikasi username/password
- Logging yang lebih detail
- Access control list (whitelist/blacklist)
- Rate limiting
- Metrics dan monitoring
- Graceful shutdown
- Configuration file

## Lisensi

MIT License
