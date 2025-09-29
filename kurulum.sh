#!/bin/bash

# Legal Documents API Kurulum Scripti
# VPS ortamında otomatik kurulum için tasarlanmıştır

set -e  # Hata durumunda scripti durdur

echo "🚀 Legal Documents API Kurulum Başlatılıyor..."

# Renklendirme için ANSI kodları
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Log fonksiyonu
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Hata kontrolü fonksiyonu
check_error() {
    if [ $? -ne 0 ]; then
        log_error "$1"
        exit 1
    fi
}

log_info "1. Dizin sahiplik ayarları yapılıyor..."
sudo chown -R $USER:$USER /opt/legal-api
check_error "Dizin sahiplik ayarları başarısız"

log_info "2. Proje dizinine geçiliyor..."
cd /opt/legal-api/GoAPIBuilder
check_error "Proje dizinine geçiş başarısız"

log_info "3. Go modülleri indiriliyor..."
go mod download
check_error "Go modül indirme başarısız"

log_info "4. Go modülleri düzenleniyor..."
go mod tidy
check_error "Go mod tidy başarısız"

log_info "5. Uygulama derleniyor (ilk build)..."
go build -o legal-api main.go
check_error "İlk derleme başarısız"

log_info "6. Optimize edilmiş uygulama derleniyor..."
go build -ldflags="-s -w" -o legal-api main.go
check_error "Optimize edilmiş derleme başarısız"

log_info "7. Çalıştırma izni veriliyor..."
chmod +x legal-api
check_error "Çalıştırma izni verme başarısız"

log_info "8. .env dosya izinleri ayarlanıyor..."
if [ -f .env ]; then
    sudo chmod 600 .env
    check_error ".env dosya izin ayarları başarısız"
else
    log_warning ".env dosyası bulunamadı, bu adım atlanıyor"
fi

log_info "9. Systemd daemon yeniden yükleniyor..."
sudo systemctl daemon-reload
check_error "Systemd daemon reload başarısız"

log_info "10. Legal API servisi etkinleştiriliyor..."
sudo systemctl enable legal-api
check_error "Servis etkinleştirme başarısız"

log_info "11. Legal API servisi başlatılıyor..."
sudo systemctl start legal-api
check_error "Servis başlatma başarısız"

log_info "12. Servis durumu kontrol ediliyor..."
sudo systemctl status legal-api --no-pager
check_error "Servis durum kontrolü başarısız"

log_info "13. Final optimize edilmiş build yapılıyor..."
go build -ldflags="-s -w" -o legal-api main.go
check_error "Final build başarısız"

log_info "14. Servis yeniden başlatılıyor..."
sudo systemctl restart legal-api
check_error "Servis restart başarısız"

log_info "15. Final servis durumu kontrol ediliyor..."
sudo systemctl status legal-api --no-pager

echo ""
echo "🎉 Kurulum tamamlandı!"
echo ""
echo "📋 Servis Bilgileri:"
echo "   • Servis adı: legal-api"
echo "   • Durumu: $(systemctl is-active legal-api)"
echo "   • Otomatik başlatma: $(systemctl is-enabled legal-api)"
echo ""
echo "🔧 Yönetim Komutları:"
echo "   • Durumu görüntüle: sudo systemctl status legal-api"
echo "   • Başlat: sudo systemctl start legal-api"
echo "   • Durdur: sudo systemctl stop legal-api"
echo "   • Yeniden başlat: sudo systemctl restart legal-api"
echo "   • Logları görüntüle: sudo journalctl -u legal-api -f"
echo ""
echo "✅ Legal Documents API başarıyla kuruldu ve çalışıyor!"