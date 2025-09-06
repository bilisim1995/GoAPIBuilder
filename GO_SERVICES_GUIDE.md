# Go Servisleri Yönetim Rehberi

## 📋 İçindekiler
- [Systemd Servis Yönetimi](#systemd-servis-yönetimi)
- [Servis Oluşturma](#servis-oluşturma)
- [Servis Yönetim Komutları](#servis-yönetim-komutları)
- [Environment Variables](#environment-variables)
- [Log Yönetimi](#log-yönetimi)
- [Nginx Proxy Yapılandırması](#nginx-proxy-yapılandırması)
- [SSL Sertifikası](#ssl-sertifikası)
- [Troubleshooting](#troubleshooting)

## 🔧 Systemd Servis Yönetimi

### Servis Dosyası Oluşturma

```bash
sudo nano /etc/systemd/system/legal-documents-api.service
```

### Örnek Servis Dosyası

```ini
[Unit]
Description=Legal Documents API Service
After=network.target
Documentation=https://github.com/yourusername/legal-documents-api

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/home/ubuntu/legal-documents-api
ExecStart=/usr/local/go/bin/go run main.go
ExecReload=/bin/kill -s HUP $MAINPID
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=legal-documents-api

# Environment Variables
Environment=PORT=8080
Environment=GIN_MODE=release
EnvironmentFile=/home/ubuntu/legal-documents-api/.env

# Security Settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/home/ubuntu/legal-documents-api

[Install]
WantedBy=multi-user.target
```

## 🚀 Servis Oluşturma

### 1. Proje Dizini Hazırlama

```bash
# Proje dizini oluştur
sudo mkdir -p /home/ubuntu/legal-documents-api
cd /home/ubuntu/legal-documents-api

# Git repository clone et
git clone https://github.com/yourusername/legal-documents-api.git .

# Dosya izinlerini düzenle
sudo chown -R ubuntu:ubuntu /home/ubuntu/legal-documents-api
chmod +x main.go
```

### 2. Environment Dosyası Oluşturma

```bash
# .env dosyası oluştur
nano /home/ubuntu/legal-documents-api/.env
```

```bash
# MongoDB Connection
MONGODB_CONNECTION_STRING=mongodb+srv://username:password@cluster.mongodb.net/
MONGODB_DATABASE=mevzuatgpt
MONGODB_METADATA_COLLECTION=metadata
MONGODB_CONTENT_COLLECTION=content

# Server Configuration
PORT=8080
GIN_MODE=release

# CORS Settings
ALLOWED_ORIGINS=https://yourdomain.com,https://www.yourdomain.com
```

### 3. Systemd Servisini Aktifleştirme

```bash
# Systemd'yi yeniden yükle
sudo systemctl daemon-reload

# Servisi etkinleştir (otomatik başlatma)
sudo systemctl enable legal-documents-api

# Servisi başlat
sudo systemctl start legal-documents-api
```

## ⚡ Servis Yönetim Komutları

### Temel Komutlar

```bash
# Servisi başlat
sudo systemctl start legal-documents-api

# Servisi durdur
sudo systemctl stop legal-documents-api

# Servisi yeniden başlat
sudo systemctl restart legal-documents-api

# Servisi yeniden yükle (config değişikliği sonrası)
sudo systemctl reload legal-documents-api

# Servis durumunu kontrol et
sudo systemctl status legal-documents-api

# Servisi devre dışı bırak (otomatik başlatmayı kapat)
sudo systemctl disable legal-documents-api

# Servisi etkinleştir (otomatik başlatmayı aç)
sudo systemctl enable legal-documents-api
```

### Servis Bilgileri

```bash
# Aktif servisleri listele
sudo systemctl list-units --type=service --state=active

# Başarısız servisleri listele
sudo systemctl list-units --type=service --state=failed

# Servis özelliklerini görüntüle
sudo systemctl show legal-documents-api

# Servisin bağımlılıklarını görüntüle
sudo systemctl list-dependencies legal-documents-api
```

## 🔍 Log Yönetimi

### Journalctl Komutları

```bash
# Tüm logları görüntüle
sudo journalctl -u legal-documents-api

# Son 100 log satırını görüntüle
sudo journalctl -u legal-documents-api -n 100

# Canlı log takibi
sudo journalctl -u legal-documents-api -f

# Belirli tarih aralığındaki loglar
sudo journalctl -u legal-documents-api --since "2024-01-01" --until "2024-01-31"

# Bugünkü loglar
sudo journalctl -u legal-documents-api --since today

# Son 1 saatteki loglar
sudo journalctl -u legal-documents-api --since "1 hour ago"

# Hata logları
sudo journalctl -u legal-documents-api -p err

# JSON formatında loglar
sudo journalctl -u legal-documents-api -o json
```

### Log Boyutu Yönetimi

```bash
# Journal boyutunu kontrol et
sudo journalctl --disk-usage

# Eski logları temizle (7 günden eski)
sudo journalctl --vacuum-time=7d

# Log boyutunu sınırla (1GB)
sudo journalctl --vacuum-size=1G
```

## 🌐 Nginx Proxy Yapılandırması

### Nginx Kurulumu

```bash
# Nginx kur
sudo apt update
sudo apt install nginx

# Nginx'i başlat ve etkinleştir
sudo systemctl start nginx
sudo systemctl enable nginx
```

### Site Yapılandırması

```bash
# Site config dosyası oluştur
sudo nano /etc/nginx/sites-available/legal-documents-api
```

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;

    # API isteklerini Go servisine yönlendir
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
        
        # CORS headers
        add_header Access-Control-Allow-Origin *;
        add_header Access-Control-Allow-Methods "GET, POST, OPTIONS";
        add_header Access-Control-Allow-Headers "Content-Type, Authorization";
        
        # Timeout settings
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://localhost:8080/health;
        access_log off;
    }

    # Static files (eğer varsa)
    location /static/ {
        alias /home/ubuntu/legal-documents-api/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### Site Aktifleştirme

```bash
# Site'ı etkinleştir
sudo ln -s /etc/nginx/sites-available/legal-documents-api /etc/nginx/sites-enabled/

# Nginx config'ini test et
sudo nginx -t

# Nginx'i yeniden başlat
sudo systemctl restart nginx
```

## 🔒 SSL Sertifikası (Let's Encrypt)

### Certbot Kurulumu

```bash
# Certbot kur
sudo apt install certbot python3-certbot-nginx

# SSL sertifikası al
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com

# Otomatik yenileme test et
sudo certbot renew --dry-run
```

### SSL Sonrası Nginx Config'i

```nginx
server {
    listen 80;
    server_name yourdomain.com www.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com www.yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
    
    location /api/ {
        proxy_pass http://localhost:8080;
        # ... diğer proxy ayarları
    }
}
```

## 🛠️ Troubleshooting

### Yaygın Sorunlar ve Çözümleri

#### 1. Servis Başlamıyor

```bash
# Servis durumunu kontrol et
sudo systemctl status legal-documents-api

# Detaylı hata mesajları
sudo journalctl -u legal-documents-api -n 50

# Go binary'nin varlığını kontrol et
which go

# Working directory'nin varlığını kontrol et
ls -la /home/ubuntu/legal-documents-api/
```

#### 2. Port Kullanımda Hatası

```bash
# Port 8080'i kullanan processleri bul
sudo lsof -i :8080

# Process'i öldür
sudo kill -9 <PID>

# Alternatif port kullan
sudo nano /etc/systemd/system/legal-documents-api.service
# Environment=PORT=8081
```

#### 3. MongoDB Bağlantı Sorunu

```bash
# Environment variables'ları kontrol et
sudo systemctl show legal-documents-api | grep Environment

# .env dosyasını kontrol et
cat /home/ubuntu/legal-documents-api/.env

# MongoDB connection test et
mongosh "your-connection-string"
```

#### 4. Nginx Proxy Sorunu

```bash
# Nginx config'ini test et
sudo nginx -t

# Nginx loglarını kontrol et
sudo tail -f /var/log/nginx/error.log

# Upstream server test et
curl http://localhost:8080/api/v1/health
```

### Performance İzleme

```bash
# CPU ve Memory kullanımı
sudo systemctl show legal-documents-api --property=CPUUsageNSec
sudo systemctl show legal-documents-api --property=MemoryCurrent

# Process details
ps aux | grep "go run main.go"

# Network connections
sudo netstat -tulpn | grep :8080
```

### Güncellemeler

```bash
# Kod güncelleme workflow'u
cd /home/ubuntu/legal-documents-api
git pull origin main
sudo systemctl restart legal-documents-api
sudo systemctl status legal-documents-api
```

### Backup ve Restore

```bash
# Servis dosyası backup
sudo cp /etc/systemd/system/legal-documents-api.service /home/ubuntu/backup/

# Environment dosyası backup
cp /home/ubuntu/legal-documents-api/.env /home/ubuntu/backup/

# Nginx config backup
sudo cp /etc/nginx/sites-available/legal-documents-api /home/ubuntu/backup/
```

## 📊 Monitoring Önerileri

1. **Log Rotation**: Logların çok büyümemesi için logrotate yapılandırın
2. **Health Checks**: Düzenli health check endpoint'leri ekleyin
3. **Alerting**: Servis down olduğunda bildirim sistemi kurun
4. **Metrics**: Prometheus/Grafana ile monitoring kurun
5. **Backup**: Düzenli config ve database backup'ları alın

Bu rehber ile Go servislerinizi Ubuntu VPS'te profesyonel şekilde yönetebilirsiniz! 🚀