package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"github.com/chromedp/chromedp"
	"golang.org/x/net/proxy"
)

// Yapılandırma işlemleri
type Config struct {
	TargetsFile string
	OutDir      string
	LogPath     string

	Workers int

	HTTPTimeout  time.Duration
	NavTimeout   time.Duration
	SettleDelay  time.Duration
	ScreenshotQ  int
	UserAgent    string
	SocksAddr    string
	ChromePath   string
	WindowWidth  int
	WindowHeight int
}

// Yardımcı fonksiyonlar

func pathCreate(path string) error {
	return os.MkdirAll(path, 0755)
}

func add2Log(logPath, msg string) {
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("[HATA] log dosyası açılamadı:", err)
		return
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	_, _ = f.WriteString(fmt.Sprintf("%s %s\n", ts, msg))
}

func readTargets(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var targets []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		targets = append(targets, line)
	}
	return targets, sc.Err()
}

func clearURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url boş")
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("desteklenmeyen protokol: %s", u.Scheme)
	}
	return u.String(), nil
}

// klasör adı üretir
func dirNameCreate(u string) string {
	u = strings.TrimSpace(u)
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimSuffix(u, "/")

	u = strings.ReplaceAll(u, "/", "_")
	u = strings.ReplaceAll(u, ":", "_")

	return u
}

// Tor HTTP client SOCKS5
func torHTTPclientCreate(socksAddr string, timeout time.Duration) (*http.Client, error) {
	dialer, err := proxy.SOCKS5("tcp", socksAddr, nil, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("SOCKS5 dialer oluşturulamadı: %w", err)
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

// chromedp'ı ayarlar
func memoryAllocator(cfg Config) (context.Context, context.CancelFunc) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("ignore-certificate-errors", true),
		chromedp.WindowSize(cfg.WindowWidth, cfg.WindowHeight),
	)

	//Ch rome yolu verilmişse kullanır
	if strings.TrimSpace(cfg.ChromePath) != "" {
		opts = append(opts, chromedp.ExecPath(cfg.ChromePath))
	}

	// SOCKS5 proxyi Chrome’a verir
	opts = append(opts, chromedp.Flag("proxy-server", "socks5://"+cfg.SocksAddr))

	return chromedp.NewExecAllocator(context.Background(), opts...)
}

// tarama aşamaları
type Result struct {
	URL        string
	OK         bool
	StatusCode int
	HTMLPath   string
	ShotPath   string
	Err        string
	Duration   time.Duration
}

func getHTML(client *http.Client, u string) (int, []byte, error) {
	resp, err := client.Get(u)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, b, nil
}

func takeScreenshot(parent context.Context, cfg Config, u string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(parent)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, cfg.NavTimeout)
	defer cancel()

	var shot []byte
	tasks := chromedp.Tasks{
		chromedp.Navigate(u),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(cfg.SettleDelay),
		chromedp.FullScreenshot(&shot, cfg.ScreenshotQ),
	}

	if err := chromedp.Run(ctx, tasks); err != nil {
		return nil, err
	}
	return shot, nil
}

func scanTarget(httpClient *http.Client, chromeAllocCtx context.Context, cfg Config, target string) Result {
	start := time.Now()
	r := Result{URL: target, OK: false}

	u, err := clearURL(target)
	if err != nil {
		r.Err = "bad url: " + err.Error()
		r.Duration = time.Since(start)
		return r
	}

	// HTML'i alır
	status, body, err := getHTML(httpClient, u)
	r.StatusCode = status
	if err != nil {
		r.Err = err.Error()
		r.Duration = time.Since(start)
		return r
	}

	siteDir := filepath.Join(cfg.OutDir, dirNameCreate(u))
	if err := pathCreate(siteDir); err != nil {
		r.Err = "create site dir: " + err.Error()
		r.Duration = time.Since(start)
		return r
	}

	// HTML'i kaydeder
	fileTS := time.Now().Format("20060102_150405")
	htmlPath := filepath.Join(siteDir, "page_"+fileTS+".html")

	if werr := os.WriteFile(htmlPath, body, 0644); werr != nil {
		r.Err = "write html: " + werr.Error()
		r.Duration = time.Since(start)
		return r
	}

	// Ekran Görğüntüsü alır
	shot, err := takeScreenshot(chromeAllocCtx, cfg, u)
	if err != nil {
		r.Err = "screenshot: " + err.Error()
		r.HTMLPath = htmlPath
		r.Duration = time.Since(start)
		return r
	}
	shotPath := filepath.Join(siteDir, "page_"+fileTS+".jpg")

	// ekran görüntüsünü kaydeder
	if werr := os.WriteFile(shotPath, shot, 0644); werr != nil {
		r.Err = "write screenshot: " + werr.Error()
		r.HTMLPath = htmlPath
		r.Duration = time.Since(start)
		return r
	}

	r.OK = true
	r.HTMLPath = htmlPath
	r.ShotPath = shotPath
	r.Duration = time.Since(start)
	return r
}

// Main fonksiyon
func main() {
	var cfg Config

	flag.StringVar(&cfg.TargetsFile, "targets", "targets.yaml", "hedeflerin yazılı olduğu dosya")
	flag.StringVar(&cfg.OutDir, "out", "output", "output klasörü")
	flag.StringVar(&cfg.LogPath, "log", "scan_report.log", "log dosyası")

	flag.IntVar(&cfg.Workers, "workers", max(2, runtime.NumCPU()/2), "parallel iş sayısı (GOroutine)")

	flag.DurationVar(&cfg.HTTPTimeout, "http-timeout", 45*time.Second, "http timeout")
	flag.DurationVar(&cfg.NavTimeout, "nav-timeout", 60*time.Second, "browser timeout ")
	flag.DurationVar(&cfg.SettleDelay, "settle", 2*time.Second, "sayfa hazırlandıktan sonraki gecikme")

	flag.IntVar(&cfg.ScreenshotQ, "quality", 85, " jpeg kalitesi 1-100")
	flag.StringVar(&cfg.UserAgent, "ua",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome Safari/537.36",
		"user-agent")

	flag.StringVar(&cfg.SocksAddr, "socks", "127.0.0.1:9150", "SOCKS5 addres (host:port)")

	flag.StringVar(&cfg.ChromePath, "chrome",
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		"chrome.exe path (windows)")

	flag.IntVar(&cfg.WindowWidth, "w", 1366, "tarayıcı ekran genişliği")
	flag.IntVar(&cfg.WindowHeight, "h", 768, "tarayıcı ekran yüksekliği")

	flag.Parse()

	runTS := time.Now().Format("2006-01-02_15-04-05")
	cfg.OutDir = filepath.Join(cfg.OutDir, "run_"+runTS)

	if err := pathCreate(cfg.OutDir); err != nil {
		fmt.Println("[HATA] run output klasoru oluşturulumadı:", err)
		return
	}

	targets, err := readTargets(cfg.TargetsFile)
	if err != nil {
		fmt.Println("[HATA] hedef dosyası okunamadı:", err)
		return
	}
	if len(targets) == 0 {
		fmt.Println("[HATA] hedef dosyası boş:")
		return
	}

	httpClient, err := torHTTPclientCreate(cfg.SocksAddr, cfg.HTTPTimeout)
	if err != nil {
		fmt.Println("[HATA]", err)
		return
	}

	allocCtx, cancelAlloc := memoryAllocator(cfg)
	defer cancelAlloc()

	startingMessage := fmt.Sprintf(
		"\n++++++++++++++++++++++++++++\n"+
			"[INFO] Tarama basladi\n"+
			"Çalışma Zamanı  : %s\n"+
			"Hedef Sayısı    : %d\n"+
			"Goroutine Sayısı: %d\n"+
			"SOCKS5 Proxy    : %s\n"+
			"++++++++++++++++++++++++++++",
		runTS, len(targets), cfg.Workers, cfg.SocksAddr,
	)

	add2Log(cfg.LogPath, startingMessage)
	fmt.Println(startingMessage)

	jobs := make(chan string)
	results := make(chan Result)

	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				fmt.Println("[INFO] Taranıyor:", t)
				results <- scanTarget(httpClient, allocCtx, cfg, t)
			}

		}()
	}
	fmt.Println()

	go func() {
		for _, t := range targets {
			jobs <- t
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	var okCount, errCount int
	for r := range results {
		if r.OK {
			okCount++
			fmt.Println("[BAŞARILI]", r.URL, "->", r.StatusCode)
			fmt.Println("HTML :", r.HTMLPath)
			fmt.Println("Görüntü :", r.ShotPath)

			add2Log(cfg.LogPath, fmt.Sprintf(
				"[Başarılı]\n"+
					"URL      : %s\n"+
					"Durum    : %d\n"+
					"HTML     : %s\n"+
					"Görüntü  : %s\n"+
					"Süre     : %s\n"+
					"****************************************",
				r.URL, r.StatusCode, r.HTMLPath, r.ShotPath, r.Duration,
			))
		} else {
			errCount++
			fmt.Println("[HATA]", r.URL)
			fmt.Println("SEBEP:", r.Err)

			add2Log(cfg.LogPath, fmt.Sprintf(
				"[HATA]\n"+
					"URL   : %s\n"+
					"Hata  : %s\n"+
					"Süre  : %s\n"+
					"****************************************",
				r.URL, r.Err, r.Duration,
			))
		}
		fmt.Println()
	}

	add2Log(cfg.LogPath, fmt.Sprintf(
		"\n++++++++++++++++++++++++++++++++++++++++"+
			"\n[INFO] Tarama tamamlandı"+
			"\nBaşarılı Sayısı : %d"+
			"\nHatalı Sayısı   : %d"+
			"\nÇıktı Dizini    : %s"+
			"\n++++++++++++++++++++++++++++++++++++++++\n",
		okCount,
		errCount,
		cfg.OutDir,
	))
	fmt.Println(
		"\n++++++++++++++++++++++++++++++++++++++++\n"+
			"[INFO] Tarama tamamlandı\n"+
			"Başarılı Sayısı : ", okCount, "\n"+
			"Hatalı Sayısı   : ", errCount, "\n"+
			"Çıktı Dizini    : ", cfg.OutDir, "\n"+
			"Log Dizini      : ", cfg.LogPath, "\n"+
			"++++++++++++++++++++++++++++++++++++++++\n",
	)

}

// goroutine sayısının en az 2 olmasını sağlar
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
