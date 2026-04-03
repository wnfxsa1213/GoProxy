package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	"goproxy/config"
	"goproxy/storage"
)

// Manager 会话租约管理器（单 mutex 保护所有租约操作）
type Manager struct {
	mu sync.Mutex

	// 内存热状态
	sessions  map[string]*Session // sessionID → Session
	byProxy   map[int64]string    // proxyID → sessionID（快速判断是否被租用）
	byTask    map[string]string   // taskID → sessionID（幂等键）
	cooldowns map[int64]int64     // proxyID → cooldown until (unix timestamp)

	store   *Store
	storage *storage.Storage
	cfg     *config.Config

	probeCandidate func(storage.Proxy) error
}

// NewManager 创建会话管理器
func NewManager(store *Store, s *storage.Storage, cfg *config.Config) *Manager {
	m := &Manager{
		sessions:       make(map[string]*Session),
		byProxy:        make(map[int64]string),
		byTask:         make(map[string]string),
		cooldowns:      make(map[int64]int64),
		store:          store,
		storage:        s,
		cfg:            cfg,
		probeCandidate: probeCandidateReachability,
	}

	// 恢复活跃会话
	active, err := store.LoadActiveSessions()
	if err == nil && len(active) > 0 {
		now := time.Now().Unix()
		restored := 0
		for _, sess := range active {
			if sess.ExpiresAt <= now {
				// 已过期，标记为 expired
				store.UpdateSessionState(sess.ID, "expired", now, "failed", false)
				continue
			}
			m.sessions[sess.ID] = sess
			m.byProxy[sess.ProxyID] = sess.ID
			m.byTask[sess.TaskID] = sess.ID
			restored++
		}
		if restored > 0 {
			log.Printf("[session] 恢复 %d 个活跃会话", restored)
		}
	}

	return m
}

// generateSessionID 生成会话 ID（16 字节 = 128 位熵）
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "sess-" + hex.EncodeToString(b)
}

// Acquire 分配独占代理会话
func (m *Manager) Acquire(req AcquireRequest) (*AcquireResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 参数默认值
	if req.TTL <= 0 {
		req.TTL = 600
	}
	if req.MinGrade == "" {
		req.MinGrade = "B"
	}
	if req.Protocol == "" {
		req.Protocol = "socks5"
	}

	// 强制冷却下限
	cfg := config.Get()
	if req.CooldownMinutes < cfg.SessionMinCooldownMin {
		req.CooldownMinutes = cfg.SessionMinCooldownMin
	}
	if req.MaxDailyUses <= 0 || req.MaxDailyUses > cfg.SessionMaxDailyUses {
		req.MaxDailyUses = cfg.SessionMaxDailyUses
	}

	// 幂等检查：task_id 已有 active session
	if req.TaskID != "" {
		if existingID, ok := m.byTask[req.TaskID]; ok {
			if existing, ok := m.sessions[existingID]; ok && existing.State == "active" && !existing.IsExpired() {
				return m.buildResponse(existing), nil
			}
		}
	}

	// 检查并发上限
	if len(m.sessions) >= cfg.SessionMaxConcurrent {
		return nil, fmt.Errorf("max_concurrent_sessions")
	}

	// 查询候选代理
	candidates, err := m.findCandidates(req)
	if err != nil || len(candidates) == 0 {
		return nil, fmt.Errorf("no_available_proxy")
	}

	selected, err := m.selectCandidate(candidates)
	if err != nil {
		return nil, err
	}

	// 创建会话
	now := time.Now().Unix()
	sess := &Session{
		ID:           generateSessionID(),
		TaskID:       req.TaskID,
		ProxyID:      selected.ID,
		ProxyAddress: selected.Address,
		Protocol:     req.Protocol,
		ExitIP:       selected.ExitIP,
		CountryCode:  selected.CountryCode,
		Timezone:     selected.Timezone,
		Grade:        selected.QualityGrade,
		State:        "active",
		Version:      1,
		LeasedAt:     now,
		ExpiresAt:    now + int64(req.TTL),
	}

	// 获取今日使用量
	usageToday, _ := m.store.IncrDailyUsage(selected.ID)
	sess.UsageToday = usageToday

	// 更新内存状态
	m.sessions[sess.ID] = sess
	m.byProxy[selected.ID] = sess.ID
	if req.TaskID != "" {
		m.byTask[req.TaskID] = sess.ID
	}

	// 持久化
	if err := m.store.SaveSession(sess); err != nil {
		log.Printf("[session] acquire: 持久化失败: %v", err)
	}

	log.Printf("[session] acquire: task=%s proxy=%s session=%s grade=%s country=%s ttl=%ds",
		req.TaskID, selected.Address, sess.ID, sess.Grade, sess.CountryCode, req.TTL)

	return m.buildResponse(sess), nil
}

// Release 释放代理会话
func (m *Manager) Release(sessionID string, result string, riskDetected bool) (*ReleaseResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		// 幂等：已释放的不报错
		return &ReleaseResponse{Released: true}, nil
	}

	if sess.State != "active" {
		return &ReleaseResponse{Released: true}, nil
	}

	now := time.Now().Unix()

	// 计算冷却时间
	cfg := config.Get()
	cooldownMin := cfg.SessionMinCooldownMin
	if riskDetected || result == "risk_blocked" {
		cooldownMin = cfg.SessionRiskCooldownMin
	}
	cooldownUntil := now + int64(cooldownMin*60)

	// 更新会话状态
	sess.State = "released"
	sess.ReleasedAt = now
	sess.Version++

	// 设置冷却
	m.cooldowns[sess.ProxyID] = cooldownUntil

	// 清理内存索引
	delete(m.byProxy, sess.ProxyID)
	delete(m.byTask, sess.TaskID)
	delete(m.sessions, sessionID)

	// 持久化
	if err := m.store.UpdateSessionState(sessionID, "released", now, result, riskDetected); err != nil {
		log.Printf("[session] release: 持久化失败: %v", err)
	}

	// 记录使用结果并重算综合质量
	newGrade := m.recordProxyOutcome(sess.ProxyID, sess.ProxyAddress, sess.Grade, result, riskDetected)

	log.Printf("[session] release: session=%s result=%s risk=%v duration=%ds grade=%s→%s cooldown=%dmin",
		sessionID, result, riskDetected, now-sess.LeasedAt, sess.Grade, newGrade, cooldownMin)

	return &ReleaseResponse{
		Released:      true,
		CooldownUntil: cooldownUntil,
		GradeUpdated:  newGrade,
	}, nil
}

// Rotate 会话内 IP 轮换
func (m *Manager) Rotate(sessionID string, reason string) (*AcquireResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session_not_found")
	}
	if sess.State != "active" {
		return nil, fmt.Errorf("session_not_active")
	}

	oldIP := sess.ExitIP
	oldProxyID := sess.ProxyID
	oldGrade := sess.Grade
	cfg := config.Get()

	// 先查找新代理（不修改任何状态）
	req := AcquireRequest{
		Protocol:        sess.Protocol,
		MinGrade:        sess.Grade,
		CooldownMinutes: cfg.SessionMinCooldownMin,
		MaxDailyUses:    cfg.SessionMaxDailyUses,
		Country:         sess.CountryCode,
	}

	// 临时排除旧代理（不修改 byProxy，只在 findCandidates 结果中过滤）
	candidates, err := m.findCandidates(req)
	if err != nil || len(candidates) == 0 {
		return nil, fmt.Errorf("no_available_proxy")
	}

	// 过滤掉当前代理
	var filtered []storage.Proxy
	for _, c := range candidates {
		if c.ID != oldProxyID {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil, fmt.Errorf("no_available_proxy")
	}

	selected := filtered[0]
	now := time.Now().Unix()

	// 确认新代理可用后，再修改旧代理状态
	m.cooldowns[oldProxyID] = now + int64(cfg.SessionRiskCooldownMin*60)
	delete(m.byProxy, oldProxyID)
	m.recordProxyOutcome(oldProxyID, sess.ProxyAddress, oldGrade, "risk_blocked", true)

	// 更新会话
	sess.ProxyID = selected.ID
	sess.ProxyAddress = selected.Address
	sess.ExitIP = selected.ExitIP
	sess.CountryCode = selected.CountryCode
	sess.Timezone = selected.Timezone
	sess.Grade = selected.QualityGrade
	sess.Version++

	m.byProxy[selected.ID] = sessionID
	usageToday, _ := m.store.IncrDailyUsage(selected.ID)
	sess.UsageToday = usageToday

	if err := m.store.SaveSession(sess); err != nil {
		log.Printf("[session] rotate: 持久化失败: %v", err)
	}

	log.Printf("[session] rotate: session=%s old=%s new=%s reason=%s",
		sessionID, oldIP, selected.ExitIP, reason)

	return m.buildResponse(sess), nil
}

// IsLeased 检查代理是否被会话占用
func (m *Manager) IsLeased(proxyID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.byProxy[proxyID]
	return ok
}

// GetLeasedIDs 获取所有被租用的代理 ID 列表
func (m *Manager) GetLeasedIDs() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]int64, 0, len(m.byProxy))
	for pid := range m.byProxy {
		ids = append(ids, pid)
	}
	return ids
}

// IsCooling 检查代理是否在冷却中
func (m *Manager) IsCooling(proxyID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	until, ok := m.cooldowns[proxyID]
	if !ok {
		return false
	}
	if time.Now().Unix() >= until {
		delete(m.cooldowns, proxyID)
		return false
	}
	return true
}

// GetActiveSessions 获取所有活跃会话
func (m *Manager) GetActiveSessions() []Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, *s)
	}
	return result
}

// GetStats 获取会话统计
func (m *Manager) GetStats() PoolSessionStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	leased := len(m.sessions)
	cooling := 0
	now := time.Now().Unix()
	for pid, until := range m.cooldowns {
		if now < until {
			cooling++
		} else {
			delete(m.cooldowns, pid)
		}
	}

	total, _ := m.storage.Count()
	available := total - leased - cooling
	if available < 0 {
		available = 0
	}

	return PoolSessionStats{
		Total:     total,
		Available: available,
		Leased:    leased,
		Cooling:   cooling,
	}
}

// ExpireCheck 扫描并释放过期会话（后台定时调用）
func (m *Manager) ExpireCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	cfg := config.Get()
	expired := 0

	for id, sess := range m.sessions {
		if now > sess.ExpiresAt {
			sess.State = "expired"
			sess.ReleasedAt = now
			sess.Version++

			// 设置冷却（过期视为 failed，使用标准冷却时间）
			m.cooldowns[sess.ProxyID] = now + int64(cfg.SessionMinCooldownMin*60)

			delete(m.byProxy, sess.ProxyID)
			delete(m.byTask, sess.TaskID)
			delete(m.sessions, id)

			if err := m.store.UpdateSessionState(id, "expired", now, "failed", false); err != nil {
				log.Printf("[session] expire: 持久化失败: %v", err)
			}
			expired++
			log.Printf("[session] expired: session=%s proxy=%s ttl=%ds",
				id, sess.ProxyAddress, sess.ExpiresAt-sess.LeasedAt)
		}
	}

	if expired > 0 {
		log.Printf("[session] 过期清理: %d 个会话", expired)
	}
}

// ReleaseAll 释放所有会话（优雅停机用）
func (m *Manager) ReleaseAll(reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().Unix()
	count := 0
	for id, sess := range m.sessions {
		sess.State = "released"
		sess.ReleasedAt = now
		if err := m.store.UpdateSessionState(id, "released", now, "failed", false); err != nil {
			log.Printf("[session] release-all: 持久化失败: %v", err)
		}
		delete(m.byProxy, sess.ProxyID)
		delete(m.byTask, sess.TaskID)
		count++
	}
	m.sessions = make(map[string]*Session)

	if count > 0 {
		log.Printf("[session] 批量释放 %d 个会话: %s", count, reason)
	}
}

// StartExpireChecker 启动过期检查后台任务
func (m *Manager) StartExpireChecker() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			m.ExpireCheck()
		}
	}()
	log.Println("[session] 过期检查器已启动（每30秒扫描）")
}

// findCandidates 查找可分配的候选代理（在 mutex 内调用）
func (m *Manager) findCandidates(req AcquireRequest) ([]storage.Proxy, error) {
	// 从 storage 查询所有可用代理
	var proxies []storage.Proxy
	var err error

	if req.Protocol != "" {
		proxies, err = m.storage.GetByProtocol(req.Protocol)
	} else {
		proxies, err = m.storage.GetAll()
	}
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	cfg := config.Get()
	cooldownSec := int64(req.CooldownMinutes * 60)
	if cooldownSec < int64(cfg.SessionMinCooldownMin*60) {
		cooldownSec = int64(cfg.SessionMinCooldownMin * 60)
	}

	var candidates []storage.Proxy
	for _, p := range proxies {
		// 排除已被租用的
		if _, leased := m.byProxy[p.ID]; leased {
			continue
		}

		// 排除评分不达标的
		if !GradeMeetsMin(p.QualityGrade, req.MinGrade) {
			continue
		}

		// 排除冷却中的（内存 + 持久化）
		if until, ok := m.cooldowns[p.ID]; ok && now < until {
			continue
		}
		lastReleased := m.store.GetLastReleasedAt(p.ID)
		if lastReleased > 0 && (now-lastReleased) < cooldownSec {
			continue
		}

		// 排除今日使用次数超限的
		usage := m.store.GetDailyUsage(p.ID)
		if usage >= req.MaxDailyUses {
			continue
		}

		// 国家匹配
		if req.Country != "" && p.CountryCode != req.Country {
			continue
		}

		candidates = append(candidates, p)
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	// 评分优先排序，同评分按延迟排序，加入随机扰动
	sort.SliceStable(candidates, func(i, j int) bool {
		return storage.CompareProxyQuality(candidates[i], candidates[j]) > 0
	})

	// 同评分内随机打散前 N 个
	if len(candidates) > 1 {
		topScore := candidates[0].QualityScore
		sameCount := 0
		for _, c := range candidates {
			if c.QualityScore == topScore {
				sameCount++
			} else {
				break
			}
		}
		if sameCount > 1 {
			// Fisher-Yates shuffle 前 sameCount 个
			for i := sameCount - 1; i > 0; i-- {
				n, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
				j := int(n.Int64())
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	return candidates, nil
}

// buildResponse 构建分配响应
func (m *Manager) buildResponse(sess *Session) *AcquireResponse {
	cfg := config.Get()
	host := cfg.SessionAdvertiseHost
	if host == "" {
		host = "127.0.0.1"
	}

	var proxyAddr string
	if sess.Protocol == "socks5" {
		port := cfg.SessionStickyPort[1:] // 去掉 ":"
		proxyAddr = fmt.Sprintf("socks5://%s:%s@%s:%s", sess.ID, "x", host, port)
	} else {
		port := cfg.SessionStickyHTTPPort[1:]
		proxyAddr = fmt.Sprintf("http://%s:%s@%s:%s", sess.ID, "x", host, port)
	}

	lastReleased := m.store.GetLastReleasedAt(sess.ProxyID)

	return &AcquireResponse{
		SessionID:      sess.ID,
		ProxyAddr:      proxyAddr,
		UpstreamIP:     sess.ExitIP,
		CountryCode:    sess.CountryCode,
		Timezone:       sess.Timezone,
		Grade:          sess.Grade,
		UsageToday:     sess.UsageToday,
		LastReleasedAt: lastReleased,
		ExpiresAt:      sess.ExpiresAt,
	}
}

func (m *Manager) selectCandidate(candidates []storage.Proxy) (storage.Proxy, error) {
	for _, candidate := range candidates {
		if err := m.probeCandidate(candidate); err != nil {
			log.Printf("[session] acquire: 跳过不可达上游 proxy=%s protocol=%s err=%v",
				candidate.Address, candidate.Protocol, err)
			if err := m.storage.IncrementFailCount(candidate.Address); err != nil {
				log.Printf("[session] acquire: 记录上游失败 proxy=%s err=%v", candidate.Address, err)
			}
			continue
		}
		return candidate, nil
	}
	return storage.Proxy{}, fmt.Errorf("no_available_proxy")
}

func probeCandidateReachability(candidate storage.Proxy) error {
	cfg := config.Get()
	timeout := 3 * time.Second
	if cfg != nil && cfg.ValidateTimeout > 0 {
		timeout = time.Duration(cfg.ValidateTimeout) * time.Second
		if timeout > 3*time.Second {
			timeout = 3 * time.Second
		}
	}

	conn, err := net.DialTimeout("tcp", candidate.Address, timeout)
	if err != nil {
		return err
	}
	return conn.Close()
}

// recordProxyOutcome 根据使用结果重算代理综合质量
func (m *Manager) recordProxyOutcome(proxyID int64, proxyAddress, currentGrade, result string, riskDetected bool) string {
	_, newGrade, err := m.storage.RecordProxyOutcome(proxyID, proxyAddress, result, riskDetected)
	if err != nil {
		log.Printf("[session] record outcome failed: proxy=%s result=%s risk=%v err=%v", proxyAddress, result, riskDetected, err)
		return currentGrade
	}
	return newGrade
}
