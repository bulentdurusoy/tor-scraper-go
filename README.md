# Tor Scraper (Go)

Bu proje, Go (Golang) programlama dili kullanılarak geliştirilmiş,
Tor ağı (.onion siteleri) üzerinde çalışan otomatik bir web tarama aracıdır.

Uygulama, hedef URL listesini bir dosyadan okuyarak sitelerin
HTML içeriklerini indirir ve sayfanın tam ekran görüntüsünü alır.
Tüm işlemler Tor Browser tarafından sağlanan SOCKS5 proxy üzerinden gerçekleştirilir.

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
├── main.go            # Ana program dosyası (Go kaynak kodu)
├── go.mod             # Go modül tanımı
├── go.sum             # Bağımlılık özetleri
├── targets.yaml       # Taranacak hedef URL listesi
├── tor-scraper.exe    # Windows için derlenmiş binary dosyası
├── tor-scraper        # Linux için derlenmiş binary dosyası
├── output/            # Program çalıştığında otomatik oluşan çıktı dizini
│   └── run_YYYY-MM-DD_HH-MM-SS/
│       └── site_adı/
│           ├── page_timestamp.html
│           └── page_timestamp.jpg
├── scan_report.log    # Tüm tarama işlemlerinin kaydedildiği log dosyası
└── README.md          # Proje açıklaması

```

```text
Program çalıştırıldığında, çıktı dosyaları otomatik olarak
`output/` dizini altında zaman damgalı (`run_YYYY-MM-DD_HH-MM-SS`)
klasörler içerisinde oluşturulur.
```

```text
Program çalıştığı süre boyunca tüm işlemler
scan_report.log adlı dosyaya otomatik olarak kaydedilir.
```

## targets.yaml Dosyası
```text
Uygulama, varsayılan olarak bulunduğu dizindeki `targets.yaml` dosyasını otomatik olarak okur.

Taramak istenilen web siteleri veya .onion adresleri,
`targets.yaml` dosyası içerisine satır satır yazılmalıdır.
Program çalıştırılırken ayrıca URL parametresi verilmesine gerek yoktur.

### Örnek `targets.yaml`
http://abc*************************************************xyz1.onion/
http://abc*************************************************xyz2.onion/
http://abc*************************************************xyz3.onion/example/path/
```

Projeyi GitHub üzerinden indirmek için:
```text
git clone https://github.com/bulentdurusoy/tor-scraper-go.git
cd tor-scraper-go
```

##Kurulum
```text
Öncelikle sistemde Go kurulu olmalıdır.

Bağımlılıkları indirmek için:
go mod tidy


Windows için derleme:
go build -o tor-scraper.exe

Linux için derleme:
GOOS=linux GOARCH=amd64 go build -o tor-scraper


Çalıştırma:

Tor Browser açık olmalı ve SOCKS5 proxy aktif olmalıdır
(varsayılan: 127.0.0.1:9150).

manuel:(proje dizininde)
go run main.go

Windows:
tor-scraper.exe

Linux:
./tor-scraper

```

Lisans:
Bu proje eğitim ve araştırma amaçlı geliştirilmiştir.
