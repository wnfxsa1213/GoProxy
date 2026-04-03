package fetcher

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"goproxy/storage"
)

// 代理来源定义
type Source struct {
	URL      string
	Protocol string // http 或 socks5
}

// 快速更新源（5-30分钟更新）- 用于紧急和补充模式
var fastUpdateSources = []Source{
	// proxifly - 每5分钟更新
	{"https://cdn.jsdelivr.net/gh/proxifly/free-proxy-list@main/proxies/protocols/http/data.txt", "http"},
	{"https://cdn.jsdelivr.net/gh/proxifly/free-proxy-list@main/proxies/protocols/socks4/data.txt", "socks5"},
	{"https://cdn.jsdelivr.net/gh/proxifly/free-proxy-list@main/proxies/protocols/socks5/data.txt", "socks5"},
	// ProxyScraper - 每30分钟更新
	{"https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/main/http.txt", "http"},
	{"https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/main/socks4.txt", "socks5"},
	{"https://raw.githubusercontent.com/ProxyScraper/ProxyScraper/main/socks5.txt", "socks5"},
	// monosans - 每小时更新
	{"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/http.txt", "http"},
	// komutan234 - 每1-2分钟更新
	{"https://raw.githubusercontent.com/komutan234/Proxy-List-Free/main/proxies/http.txt", "http"},
	{"https://raw.githubusercontent.com/komutan234/Proxy-List-Free/main/proxies/socks5.txt", "socks5"},
	// iplocate - 每30分钟更新
	{"https://raw.githubusercontent.com/iplocate/free-proxy-list/main/protocols/http.txt", "http"},
	{"https://raw.githubusercontent.com/iplocate/free-proxy-list/main/protocols/socks5.txt", "socks5"},
}

// 慢速更新源（每天更新）- 用于优化轮换模式
var slowUpdateSources = []Source{
	// TheSpeedX - 每天更新
	{"https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/http.txt", "http"},
	{"https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/socks4.txt", "socks5"},
	{"https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/socks5.txt", "socks5"},
	// monosans SOCKS
	{"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks4.txt", "socks5"},
	{"https://raw.githubusercontent.com/monosans/proxy-list/main/proxies/socks5.txt", "socks5"},
	// databay-labs - 备用源
	{"https://cdn.jsdelivr.net/gh/databay-labs/free-proxy-list/http.txt", "http"},
	{"https://cdn.jsdelivr.net/gh/databay-labs/free-proxy-list/socks5.txt", "socks5"},
}

// 所有源
var allSources = append(fastUpdateSources, slowUpdateSources...)

type Fetcher struct {
	sources       []Source
	client        *http.Client
	sourceManager *SourceManager
}

func New(httpURL, socks5URL string, sourceManager *SourceManager) *Fetcher {
	return &Fetcher{
		sources:       allSources,
		sourceManager: sourceManager,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchSmart 智能抓取：根据模式和协议需求选择源
func (f *Fetcher) FetchSmart(mode string, preferredProtocol string) ([]storage.Proxy, error) {
	var sources []Source

	switch mode {
	case "emergency":
		// 紧急模式：忽略断路器，强制使用所有源（包括被禁用的）
		sources = f.filterAvailableSources(allSources, preferredProtocol, true)
		log.Printf("[fetch] 🚨 紧急模式: 使用 %d 个源（忽略断路器）", len(sources))

	case "refill":
		// 补充模式：使用快更新源
		sources = f.filterAvailableSources(fastUpdateSources, preferredProtocol, false)
		log.Printf("[fetch] 🔄 补充模式: 使用 %d 个快更新源", len(sources))

	case "optimize":
		// 优化模式：随机选择2-3个慢更新源
		sources = f.selectRandomSources(slowUpdateSources, 3, preferredProtocol)
		log.Printf("[fetch] ⚡ 优化模式: 使用 %d 个源", len(sources))

	default:
		sources = f.filterAvailableSources(fastUpdateSources, preferredProtocol, false)
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no available sources")
	}

	return f.fetchFromSources(sources)
}

// filterAvailableSources 过滤可用的源（通过断路器）
// ignoreCircuitBreaker: 是否忽略断路器（Emergency 模式下使用）
func (f *Fetcher) filterAvailableSources(sources []Source, preferredProtocol string, ignoreCircuitBreaker bool) []Source {
	var available []Source
	for _, src := range sources {
		// 检查断路器（紧急模式下忽略）
		if !ignoreCircuitBreaker && f.sourceManager != nil && !f.sourceManager.CanUseSource(src.URL) {
			continue
		}
		// 如果指定了协议偏好，优先该协议的源
		if preferredProtocol != "" && src.Protocol != "" && src.Protocol != preferredProtocol {
			continue
		}
		available = append(available, src)
	}
	return available
}

// selectRandomSources 随机选择N个源
func (f *Fetcher) selectRandomSources(sources []Source, count int, preferredProtocol string) []Source {
	available := f.filterAvailableSources(sources, preferredProtocol, false)
	if len(available) <= count {
		return available
	}

	// 随机打乱
	shuffled := make([]Source, len(available))
	copy(shuffled, available)
	for i := range shuffled {
		j := i + int(time.Now().UnixNano())%(len(shuffled)-i)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

// fetchFromSources 从指定源列表抓取
func (f *Fetcher) fetchFromSources(sources []Source) ([]storage.Proxy, error) {
	type result struct {
		proxies []storage.Proxy
		source  Source
		err     error
	}

	ch := make(chan result, len(sources))
	for _, src := range sources {
		go func(s Source) {
			proxies, err := f.fetchFromURL(s.URL, s.Protocol)
			ch <- result{proxies: proxies, source: s, err: err}
		}(src)
	}

	var all []storage.Proxy
	seen := make(map[string]bool)
	for range sources {
		r := <-ch
		if r.err != nil {
			log.Printf("[fetch] ❌ %s error: %v", r.source.URL, r.err)
			if f.sourceManager != nil {
				f.sourceManager.RecordFail(r.source.URL, 3, 5, 30)
			}
			continue
		}

		// 记录成功
		if f.sourceManager != nil {
			f.sourceManager.RecordSuccess(r.source.URL)
		}

		// 去重
		var deduped []storage.Proxy
		for _, p := range r.proxies {
			if !seen[p.Address] {
				seen[p.Address] = true
				deduped = append(deduped, p)
			}
		}
		log.Printf("[fetch] ✅ %d 个 %s 代理 from %s", len(deduped), r.source.Protocol, r.source.URL)
		all = append(all, deduped...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no proxies fetched")
	}
	log.Printf("[fetch] 总共抓取: %d 个代理（去重后）", len(all))
	return all, nil
}

// Fetch 从所有来源并发抓取代理
func (f *Fetcher) Fetch() ([]storage.Proxy, error) {
	type result struct {
		proxies []storage.Proxy
		source  Source
		err     error
	}

	ch := make(chan result, len(f.sources))
	for _, src := range f.sources {
		go func(s Source) {
			proxies, err := f.fetchFromURL(s.URL, s.Protocol)
			ch <- result{proxies: proxies, source: s, err: err}
		}(src)
	}

	var all []storage.Proxy
	seen := make(map[string]bool)
	for range f.sources {
		r := <-ch
		if r.err != nil {
			log.Printf("fetch %s error: %v", r.source.URL, r.err)
			continue
		}
		// 去重
		var deduped []storage.Proxy
		for _, p := range r.proxies {
			if !seen[p.Address] {
				seen[p.Address] = true
				deduped = append(deduped, p)
			}
		}
		log.Printf("fetched %d %s proxies from %s", len(deduped), r.source.Protocol, r.source.URL)
		all = append(all, deduped...)
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no proxies fetched")
	}
	log.Printf("total fetched: %d proxies (deduped)", len(all))
	return all, nil
}

func (f *Fetcher) fetchFromURL(url, protocol string) ([]storage.Proxy, error) {
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	return parseProxyList(resp.Body, protocol)
}

func parseProxyList(r io.Reader, protocol string) ([]storage.Proxy, error) {
	var proxies []storage.Proxy
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		addr := line
		proto := protocol
		// 支持 protocol://host:port 格式
		if idx := strings.Index(line, "://"); idx != -1 {
			proto = line[:idx]
			addr = line[idx+3:]
			// socks4 当 socks5 处理
			if proto == "socks4" {
				proto = "socks5"
			}
		}
		parts := strings.Split(addr, ":")
		if len(parts) != 2 {
			continue
		}
		proxies = append(proxies, storage.Proxy{
			Address:  addr,
			Protocol: proto,
		})
	}
	return proxies, scanner.Err()
}
