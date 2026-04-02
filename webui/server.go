package webui

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"goproxy/config"
	"goproxy/logger"
	"goproxy/pool"
	"goproxy/storage"
	"goproxy/validator"
)

// 简单内存 session
var (
	sessions   = make(map[string]time.Time)
	sessionsMu sync.Mutex
)

func newSession() string {
	token := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))))
	sessionsMu.Lock()
	sessions[token] = time.Now().Add(24 * time.Hour)
	sessionsMu.Unlock()
	return token
}

func validSession(r *http.Request) bool {
	cookie, err := r.Cookie("session")
	if err != nil {
		return false
	}
	sessionsMu.Lock()
	expiry, ok := sessions[cookie.Value]
	sessionsMu.Unlock()
	return ok && time.Now().Before(expiry)
}

type FetchTrigger func()

type Server struct {
	storage       *storage.Storage
	cfg           *config.Config
	poolMgr       *pool.Manager
	fetchTrigger  FetchTrigger
	configChanged chan<- struct{}
}

func New(s *storage.Storage, cfg *config.Config, pm *pool.Manager, ft FetchTrigger, cc chan<- struct{}) *Server {
	return &Server{
		storage:       s,
		cfg:           cfg,
		poolMgr:       pm,
		fetchTrigger:  ft,
		configChanged: cc,
	}
}

// RouteRegistrar 外部路由注册函数
type RouteRegistrar func(mux *http.ServeMux)

func (s *Server) Start(registrars ...RouteRegistrar) {
	mux := http.NewServeMux()

	// 添加日志中间件
	loggedMux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[webui] %s %s | Host: %s | RemoteAddr: %s",
			r.Method, r.URL.Path, r.Host, r.RemoteAddr)
		mux.ServeHTTP(w, r)
	})

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/login", s.handleLogin)
	mux.HandleFunc("/logout", s.handleLogout)

	// 只读 API（访客可访问）
	mux.HandleFunc("/api/stats", s.readOnlyMiddleware(s.apiStats))
	mux.HandleFunc("/api/proxies", s.readOnlyMiddleware(s.apiProxies))
	mux.HandleFunc("/api/logs", s.readOnlyMiddleware(s.apiLogs))
	mux.HandleFunc("/api/pool/status", s.readOnlyMiddleware(s.apiPoolStatus))
	mux.HandleFunc("/api/pool/quality", s.readOnlyMiddleware(s.apiQualityDistribution))
	mux.HandleFunc("/api/config", s.readOnlyMiddleware(s.apiConfig))
	mux.HandleFunc("/api/auth/check", s.apiAuthCheck) // 检查登录状态

	// 管理员 API（需要登录）
	mux.HandleFunc("/api/proxy/delete", s.authMiddleware(s.apiDeleteProxy))
	mux.HandleFunc("/api/proxy/refresh", s.authMiddleware(s.apiRefreshProxy))
	mux.HandleFunc("/api/fetch", s.authMiddleware(s.apiFetch))
	mux.HandleFunc("/api/refresh-latency", s.authMiddleware(s.apiRefreshLatency))
	mux.HandleFunc("/api/config/save", s.authMiddleware(s.apiConfigSave))

	// 注册外部路由（如 Session API）
	for _, reg := range registrars {
		reg(mux)
	}

	log.Printf("WebUI listening on %s", s.cfg.WebUIPort)
	go func() {
		if err := http.ListenAndServe(s.cfg.WebUIPort, loggedMux); err != nil {
			log.Fatalf("webui: %v", err)
		}
	}()
}

// authMiddleware 管理员权限中间件（必须登录）
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validSession(r) {
			if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
				jsonError(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

// readOnlyMiddleware 只读中间件（访客可访问，但会标记是否为管理员）
func (s *Server) readOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 访客和管理员都可以访问，通过 validSession 判断权限
		next(w, r)
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	// 允许访客访问（只读模式），管理员登录后有完整权限
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, dashboardHTML)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, loginHTML)
		return
	}
	password := r.FormValue("password")
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(password)))
	if hash != s.cfg.WebUIPasswordHash {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, loginHTMLWithError)
		return
	}
	token := newSession()
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil {
		sessionsMu.Lock()
		delete(sessions, cookie.Value)
		sessionsMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusFound)
}

// apiAuthCheck 检查当前用户是否为管理员
func (s *Server) apiAuthCheck(w http.ResponseWriter, r *http.Request) {
	isAdmin := validSession(r)
	jsonOK(w, map[string]interface{}{
		"isAdmin": isAdmin,
		"mode":    func() string {
			if isAdmin {
				return "admin"
			}
			return "guest"
		}(),
	})
}

func (s *Server) apiStats(w http.ResponseWriter, r *http.Request) {
	total, _ := s.storage.Count()
	httpCount, _ := s.storage.CountByProtocol("http")
	socks5Count, _ := s.storage.CountByProtocol("socks5")
	jsonOK(w, map[string]interface{}{
		"total":  total,
		"http":   httpCount,
		"socks5": socks5Count,
		"port":   s.cfg.ProxyPort,
	})
}

func (s *Server) apiProxies(w http.ResponseWriter, r *http.Request) {
	protocol := r.URL.Query().Get("protocol")
	var proxies []storage.Proxy
	var err error
	if protocol != "" {
		proxies, err = s.storage.GetByProtocol(protocol)
	} else {
		proxies, err = s.storage.GetAll()
	}
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, proxies)
}

func (s *Server) apiDeleteProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	s.storage.Delete(req.Address)
	jsonOK(w, map[string]string{"status": "deleted"})
}

func (s *Server) apiRefreshProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Address string `json:"address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 从数据库获取代理信息
	proxies, err := s.storage.GetAll()
	if err != nil {
		jsonError(w, "failed to get proxy", http.StatusInternalServerError)
		return
	}
	
	var targetProxy *storage.Proxy
	for i := range proxies {
		if proxies[i].Address == req.Address {
			targetProxy = &proxies[i]
			break
		}
	}
	
	if targetProxy == nil {
		jsonError(w, "proxy not found", http.StatusNotFound)
		return
	}

	// 异步验证并更新
	go func() {
		cfg := config.Get()
		v := validator.New(1, cfg.ValidateTimeout, cfg.ValidateURL)
		
		log.Printf("[webui] refreshing proxy: %s", req.Address)
		valid, latency, exitIP, exitLocation, countryCode, tz := v.ValidateOne(*targetProxy)

		if valid {
			latencyMs := int(latency.Milliseconds())
			s.storage.UpdateExitInfo(req.Address, exitIP, exitLocation, latencyMs, countryCode, tz)
			log.Printf("[webui] proxy refreshed: %s latency=%dms grade=%s", req.Address, latencyMs, storage.CalculateQualityGrade(latencyMs))
		} else {
			s.storage.Delete(req.Address)
			log.Printf("[webui] proxy validation failed, removed: %s", req.Address)
		}
	}()

	jsonOK(w, map[string]string{"status": "refresh started"})
}

func (s *Server) apiFetch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go s.fetchTrigger()
	jsonOK(w, map[string]string{"status": "fetch started"})
}

func (s *Server) apiRefreshLatency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go func() {
		log.Println("[webui] refreshing latency for all proxies...")
		proxies, err := s.storage.GetAll()
		if err != nil {
			log.Printf("[webui] get proxies error: %v", err)
			return
		}
		if len(proxies) == 0 {
			log.Println("[webui] no proxies to refresh")
			return
		}

		cfg := config.Get()
		validate := validator.New(cfg.ValidateConcurrency, cfg.ValidateTimeout, cfg.ValidateURL)

		log.Printf("[webui] refreshing latency for %d proxies...", len(proxies))
		updated := 0
		for r := range validate.ValidateStream(proxies) {
			if r.Valid {
				latencyMs := int(r.Latency.Milliseconds())
				s.storage.UpdateExitInfo(r.Proxy.Address, r.ExitIP, r.ExitLocation, latencyMs, r.CountryCode, r.Timezone)
				updated++
			} else {
				s.storage.Delete(r.Proxy.Address)
			}
		}
		log.Printf("[webui] latency refresh done: updated=%d", updated)
	}()
	jsonOK(w, map[string]string{"status": "refresh started"})
}

func (s *Server) apiLogs(w http.ResponseWriter, r *http.Request) {
	lines := logger.GetLines(100)
	jsonOK(w, map[string]interface{}{"lines": lines})
}

// apiConfig 获取配置
func (s *Server) apiConfig(w http.ResponseWriter, r *http.Request) {
	cfg := config.Get()
	httpSlots, socks5Slots := cfg.CalculateSlots()
	
	jsonOK(w, map[string]interface{}{
		// 池子配置
		"pool_max_size":        cfg.PoolMaxSize,
		"pool_http_ratio":      cfg.PoolHTTPRatio,
		"pool_min_per_protocol": cfg.PoolMinPerProtocol,
		"pool_http_slots":      httpSlots,
		"pool_socks5_slots":    socks5Slots,
		
		// 延迟配置
		"max_latency_ms":         cfg.MaxLatencyMs,
		"max_latency_emergency":  cfg.MaxLatencyEmergency,
		"max_latency_healthy":    cfg.MaxLatencyHealthy,
		
		// 验证配置
		"validate_concurrency":   cfg.ValidateConcurrency,
		"validate_timeout":       cfg.ValidateTimeout,
		
		// 健康检查配置
		"health_check_interval":  cfg.HealthCheckInterval,
		"health_check_batch_size": cfg.HealthCheckBatchSize,
		
		// 优化配置
		"optimize_interval":      cfg.OptimizeInterval,
		"replace_threshold":      cfg.ReplaceThreshold,
	})
}

// apiConfigSave 保存配置
func (s *Server) apiConfigSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PoolMaxSize           int     `json:"pool_max_size"`
		PoolHTTPRatio         float64 `json:"pool_http_ratio"`
		PoolMinPerProtocol    int     `json:"pool_min_per_protocol"`
		MaxLatencyMs          int     `json:"max_latency_ms"`
		MaxLatencyEmergency   int     `json:"max_latency_emergency"`
		MaxLatencyHealthy     int     `json:"max_latency_healthy"`
		ValidateConcurrency   int     `json:"validate_concurrency"`
		ValidateTimeout       int     `json:"validate_timeout"`
		HealthCheckInterval   int     `json:"health_check_interval"`
		HealthCheckBatchSize  int     `json:"health_check_batch_size"`
		OptimizeInterval      int     `json:"optimize_interval"`
		ReplaceThreshold      float64 `json:"replace_threshold"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 验证配置有效性
	if req.PoolMaxSize <= 0 || req.PoolHTTPRatio <= 0 || req.PoolHTTPRatio > 1 {
		jsonError(w, "invalid pool config", http.StatusBadRequest)
		return
	}

	// 记录旧配置
	oldCfg := config.Get()
	oldSize := oldCfg.PoolMaxSize
	oldRatio := oldCfg.PoolHTTPRatio

	// 更新配置
	newCfg := *oldCfg
	newCfg.PoolMaxSize = req.PoolMaxSize
	newCfg.PoolHTTPRatio = req.PoolHTTPRatio
	newCfg.PoolMinPerProtocol = req.PoolMinPerProtocol
	newCfg.MaxLatencyMs = req.MaxLatencyMs
	newCfg.MaxLatencyEmergency = req.MaxLatencyEmergency
	newCfg.MaxLatencyHealthy = req.MaxLatencyHealthy
	newCfg.ValidateConcurrency = req.ValidateConcurrency
	newCfg.ValidateTimeout = req.ValidateTimeout
	newCfg.HealthCheckInterval = req.HealthCheckInterval
	newCfg.HealthCheckBatchSize = req.HealthCheckBatchSize
	newCfg.OptimizeInterval = req.OptimizeInterval
	newCfg.ReplaceThreshold = req.ReplaceThreshold

	if err := config.Save(&newCfg); err != nil {
		jsonError(w, "save config error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 通知配置变更
	select {
	case s.configChanged <- struct{}{}:
	default:
	}

	// 如果池子大小或比例变更，调整池子
	if oldSize != req.PoolMaxSize || oldRatio != req.PoolHTTPRatio {
		go s.poolMgr.AdjustForConfigChange(oldSize, oldRatio)
	}

	log.Printf("[config] 配置已更新: 池子=%d HTTP=%.0f%% 延迟=%dms",
		req.PoolMaxSize, req.PoolHTTPRatio*100, req.MaxLatencyMs)
	jsonOK(w, map[string]string{"status": "saved"})
}

// apiPoolStatus 获取池子状态
func (s *Server) apiPoolStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.poolMgr.GetStatus()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, status)
}

// apiQualityDistribution 获取质量分布
func (s *Server) apiQualityDistribution(w http.ResponseWriter, r *http.Request) {
	dist, err := s.storage.GetQualityDistribution()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, dist)
}

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
