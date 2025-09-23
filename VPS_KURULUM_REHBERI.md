# 🚀 Legal Documents API - VPS Kurulum Rehberi

## 📋 İçindekiler
- [Sistem Gereksinimleri](#sistem-gereksinimleri)
- [Go Kurulumu](#go-kurulumu)
- [Proje Kurulumu](#proje-kurulumu)
- [Environment Variables](#environment-variables)
- [Systemd Servis Yapılandırması](#systemd-servis-yapılandırması)
- [Nginx Yapılandırması](#nginx-yapılandırması)
- [SSL Sertifikası](#ssl-sertifikası)
- [Servis Yönetimi](#servis-yönetimi)
- [Troubleshooting](#troubleshooting)

---

## 🖥️ Sistem Gereksinimleri

### Minimum Donanım
- **RAM:** 2GB (4GB önerilen)
- **CPU:** 2 Core (4 Core önerilen)
- **Disk:** 20GB SSD
- **OS:** Ubuntu 20.04+ / CentOS 8+ / Debian 11+

### Yazılım Gereksinimleri
- **Go:** 1.19+
- **Nginx:** 1.18+ (kurulu varsayılmıştır)
- **Git:** Latest
- **MongoDB:** Atlas Cloud (uzak bağlantı)

---

## 🔧 Go Kurulumu

### Go 1.19+ Kurulumu
```bash
# Go'nun güncel sürümünü indir
cd /tmp
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz

# Eski Go sürümünü kaldır (varsa)
sudo rm -rf /usr/local/go

# Yeni Go sürümünü kur
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# PATH'e ekle
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Kurulumu doğrula
go version
```

---

## 📂 Proje Kurulumu

### 1. Proje Dizini Oluşturma
```bash
# Proje dizini oluştur
sudo mkdir -p /opt/legal-documents-api
cd /opt/legal-documents-api

# Git repository clone et
git clone <REPO_URL> .

# Dosya sahipliklerini ayarla
sudo chown -R $USER:$USER /opt/legal-documents-api
```

### 2. Dependencies Kurulumu
```bash
cd /opt/legal-documents-api

# Go dependencies'i yükle
go mod download
go mod tidy

# Build test et
go build -o legal-documents-api main.go
```

### 3. Binary Oluşturma
```bash
# Production binary oluştur
go build -ldflags="-s -w" -o legal-documents-api main.go

# Executable yap
chmod +x legal-documents-api

# Binary'yi test et
./legal-documents-api --help
```

---

## 🔐 Environment Variables

### 1. Environment Dosyası Oluşturma
```bash
# Environment dosyası oluştur
sudo nano /opt/legal-documents-api/.env
```

### 2. Environment Değişkenleri
```bash
# MongoDB Bağlantısı
MONGODB_CONNECTION_STRING=mongodb+srv://username:password@cluster.mongodb.net/?retryWrites=true&w=majority
MONGODB_DATABASE=mevzuatgpt
MONGODB_METADATA_COLLECTION=metadata
MONGODB_CONTENT_COLLECTION=content

# Server Yapılandırması
PORT=8080
GIN_MODE=release

# API Kimlik Doğrulama
API_USERNAME=admin
API_PASSWORD=your_secure_password_here

# CORS Ayarları
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com

# Logging
LOG_LEVEL=info
```

### 3. Dosya İzinleri
```bash
# .env dosyasının güvenliğini sağla
sudo chmod 600 /opt/legal-documents-api/.env
sudo chown root:root /opt/legal-documents-api/.env
```

---

## ⚙️ Systemd Servis Yapılandırması

### 1. Servis Dosyası Oluşturma
```bash
sudo nano /etc/systemd/system/legal-documents-api.service
```

### 2. Servis Dosyası İçeriği
```ini
[Unit]
Description=Legal Documents API Service
After=network.target mongodb.service
Documentation=https://github.com/yourusername/legal-documents-api
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/opt/legal-documents-api
ExecStart=/opt/legal-documents-api/legal-documents-api
ExecReload=/bin/kill -s HUP $MAINPID
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=legal-documents-api

# Environment Variables
Environment=PORT=8080
Environment=GIN_MODE=release
EnvironmentFile=/opt/legal-documents-api/.env

# Security Settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/legal-documents-api
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE
SecureBits=keep-caps

# Resource Limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
```

### 3. Servisi Etkinleştirme
```bash
# Systemd'yi yeniden yükle
sudo systemctl daemon-reload

# Servisi etkinleştir (otomatik başlatma)
sudo systemctl enable legal-documents-api

# Servisi başlat
sudo systemctl start legal-documents-api

# Durumu kontrol et
sudo systemctl status legal-documents-api
```

---

## 🌐 Nginx Yapılandırması

### 1. Site Yapılandırması (Nginx zaten kurulu)
```bash
# Site config dosyasını düzenle
sudo nano /etc/nginx/sites-available/legal-documents-api
```

### 2. Nginx Virtual Host
```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;

    # API endpoints
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # CORS headers (ek güvenlik)
        add_header Access-Control-Allow-Origin "*" always;
        add_header Access-Control-Allow-Methods "GET, POST, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization" always;
        
        # Timeout settings
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://localhost:8080/api/v1/health;
        access_log off;
    }

    # Root endpoint
    location / {
        proxy_pass http://localhost:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types
        application/json
        application/javascript
        text/css
        text/xml
        text/plain;
}
```

### 3. Site'ı Aktifleştirme
```bash
# Site'ı etkinleştir
sudo ln -sf /etc/nginx/sites-available/legal-documents-api /etc/nginx/sites-enabled/

# Default site'ı kaldır (opsiyonel)
sudo rm -f /etc/nginx/sites-enabled/default

# Nginx config'ini test et
sudo nginx -t

# Nginx'i yeniden başlat
sudo systemctl restart nginx
```

---

## 🔒 SSL Sertifikası (Let's Encrypt)

### 1. Certbot Kurulumu
```bash
# Certbot kur
sudo apt update
sudo apt install certbot python3-certbot-nginx -y
```

### 2. SSL Sertifikası Alma
```bash
# SSL sertifikası al
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Otomatik yenileme test et
sudo certbot renew --dry-run
```

### 3. Cron Job (Otomatik Yenileme)
```bash
# Cron job ekle
sudo crontab -e

# Aşağıdaki satırı ekle:
0 12 * * * /usr/bin/certbot renew --quiet
```

---

## ⚡ Servis Yönetimi

### Temel Komutlar
```bash
# Servisi başlat
sudo systemctl start legal-documents-api

# Servisi durdur
sudo systemctl stop legal-documents-api

# Servisi yeniden başlat
sudo systemctl restart legal-documents-api

# Servis durumunu kontrol et
sudo systemctl status legal-documents-api

# Canlı log takibi
sudo journalctl -u legal-documents-api -f

# Son 100 log satırı
sudo journalctl -u legal-documents-api -n 100
```

### Güncellemeler
```bash
# Kod güncelleme workflow'u
cd /opt/legal-documents-api
git pull origin main
go build -ldflags="-s -w" -o legal-documents-api main.go
sudo systemctl restart legal-documents-api
sudo systemctl status legal-documents-api
```

---

## 🛠️ Troubleshooting

### 1. Servis Başlamıyor
```bash
# Detaylı hata mesajları
sudo journalctl -u legal-documents-api -n 50

# Go binary'nin varlığını kontrol et
which go
ls -la /opt/legal-documents-api/legal-documents-api

# Environment variables'ları kontrol et
sudo systemctl show legal-documents-api | grep Environment
```

### 2. Port Kullanımda Hatası
```bash
# Port 8080'i kullanan processleri bul
sudo lsof -i :8080

# Process'i öldür
sudo kill -9 <PID>
```

### 3. MongoDB Bağlantı Sorunu
```bash
# Environment variables'ları kontrol et
sudo cat /opt/legal-documents-api/.env

# MongoDB connection test et
./legal-documents-api # Manuel başlatma ve log kontrol
```

### 4. Nginx Proxy Sorunu
```bash
# Nginx config'ini test et
sudo nginx -t

# Nginx loglarını kontrol et
sudo tail -f /var/log/nginx/error.log

# Upstream server test et
curl http://localhost:8080/api/v1/health
```

### 5. SSL Sorunları
```bash
# Certbot logları
sudo tail -f /var/log/letsencrypt/letsencrypt.log

# SSL sertifika durumu
sudo certbot certificates
```

---

## 📊 Monitoring ve Backup

### Log Rotation
```bash
# Logrotate config
sudo nano /etc/logrotate.d/legal-documents-api

/var/log/legal-documents-api/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    create 0644 ubuntu ubuntu
    postrotate
        systemctl reload legal-documents-api
    endscript
}
```

### Backup Script
```bash
# Backup script oluştur
sudo nano /opt/legal-documents-api/backup.sh

#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/opt/backups/legal-documents-api"

# Backup directory oluştur
mkdir -p $BACKUP_DIR

# Config dosyalarını backup al
tar -czf $BACKUP_DIR/config_$DATE.tar.gz \
    /opt/legal-documents-api/.env \
    /etc/systemd/system/legal-documents-api.service \
    /etc/nginx/sites-available/legal-documents-api

# Binary backup
cp /opt/legal-documents-api/legal-documents-api $BACKUP_DIR/binary_$DATE

echo "Backup completed: $BACKUP_DIR"
```

---

## 🎯 Final Checklist

- [ ] Go 1.19+ kurulu ve çalışıyor
- [ ] Proje `/opt/legal-documents-api` dizininde
- [ ] Dependencies yüklü ve binary oluşturuldu
- [ ] Environment variables `.env` dosyasında tanımlı
- [ ] Systemd servisi etkinleştirildi ve çalışıyor
- [ ] Nginx proxy yapılandırması aktif
- [ ] SSL sertifikası kurulu (HTTPS)
- [ ] Firewall 80, 443 portları açık
- [ ] Backup ve monitoring scriptleri yerinde

## 🚀 Test

```bash
# API health check
curl https://yourdomain.com/api/v1/health

# Kurum duyuru endpoint test
curl "https://yourdomain.com/api/v1/kurum-duyuru?kurum_id=68bf0cd13907e0d3ac876705"

# Documents endpoint test
curl "https://yourdomain.com/api/v1/documents?kurum_id=68bd76d0f639e817a373d15e&limit=5"
```

---

**🎉 Kurulum tamamlandı! Legal Documents API artık production'da çalışıyor.**