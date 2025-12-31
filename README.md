# Tor Scraper (Go)

Bu proje, **Go (Golang)** programlama dili kullanılarak geliştirilmiş,  
**Tor ağı (.onion siteleri)** üzerinde çalışan otomatik bir web tarama aracıdır.

Uygulama; hedef URL listesini bir dosyadan okuyarak web sitelerinin:

- HTML içeriklerini indirir  
- Sayfanın **tam ekran görüntüsünü (screenshot)** alır  

Tüm işlemler, **Tor Browser tarafından sağlanan SOCKS5 proxy** üzerinden gerçekleştirilir.

---

## Özellikler

- Tor ağı (.onion) üzerinden web sitelerini tarama  
- SOCKS5 proxy desteği (Tor Browser uyumlu)  
- Hedef URL’leri dosyadan okuma (`targets.yaml`)  
- URL doğrulama ve düzenleme  
- HTML içeriğini dosyaya kaydetme  
- Web sayfasının tam ekran görüntüsünü alma  
- Site bazlı klasörleme  
- Zaman damgalı (timestamp) çıktı dosyaları  
- Eş zamanlı tarama (goroutine kullanımı)  
- Hata yönetimi ve detaylı loglama  
- Windows ve Linux için derlenebilir binary desteği  

---

## Kullanılan Teknolojiler

- Go (Golang)  
- net/http  
- chromedp  
- golang.org/x/net/proxy  
- Tor Browser (SOCKS5 proxy)  

---

## Proje Dosya Yapısı

```text
tor-scraper-go/
├── main.go          # Ana program dosyası
├── go.mod           # Go modül tanımı
├── go.sum           # Bağımlılık özetleri
├── targets.yaml     # Temsili hedef listesi
├── tor-scraper      # Linux binary
├── tor-scraper.exe  # Windows binary
└── README.md        # Proje açıklaması
