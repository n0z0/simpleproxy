# Simple SOCKS5 Proxy Server

Server proxy SOCKS5 sederhana yang ditulis dalam Go.

## Fitur

- ✅ Implementasi protokol SOCKS5 dasar
- ✅ Mendukung koneksi TCP (CONNECT command)
- ✅ Mendukung relay UDP (UDP ASSOCIATE command)
- ✅ Mendukung alamat IPv4, IPv6, dan domain name
- ✅ Tanpa autentikasi (no-auth mode)
- ✅ Relay data dua arah antara client dan target
- ✅ Server listen pada TCP dan UDP di port yang sama

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

### TCP Functions
- `handshake()` - Menangani SOCKS5 handshake awal
- `handleRequest()` - Memproses request koneksi (CONNECT atau UDP ASSOCIATE)
- `sendReply()` - Mengirim response ke client
- `relay()` - Meneruskan data TCP antara client dan target server

### UDP Functions
- `handleUDPAssociate()` - Menangani UDP ASSOCIATE request
- `handleUDPRelay()` - Loop utama untuk menerima paket UDP
- `processUDPPacket()` - Parse dan forward paket UDP individual
- `buildUDPReply()` - Membuat SOCKS5 UDP reply header

## Limitasi

- Hanya mendukung command CONNECT dan UDP ASSOCIATE (tidak mendukung BIND)
- Tidak ada autentikasi (semua koneksi diterima)
- Tidak ada logging detail atau metrics
- Tidak ada rate limiting atau access control
- UDP fragmentation tidak didukung

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
