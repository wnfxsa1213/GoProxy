package storage

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Proxy struct {
	ID           int64     `json:"id"`
	Address      string    `json:"address"`
	Protocol     string    `json:"protocol"`
	ExitIP       string    `json:"exit_ip"`
	ExitLocation string    `json:"exit_location"`
	CountryCode  string    `json:"country_code"`
	Timezone     string    `json:"timezone"`
	Latency      int       `json:"latency"`
	QualityGrade string    `json:"quality_grade"`
	QualityScore int       `json:"quality_score"`
	RiskCount    int       `json:"risk_count"`
	UseCount     int       `json:"use_count"`
	SuccessCount int       `json:"success_count"`
	FailCount    int       `json:"fail_count"`
	LastUsed     time.Time `json:"last_used"`
	LastCheck    time.Time `json:"last_check"`
	CreatedAt    time.Time `json:"created_at"`
	Status       string    `json:"status"`
}

// SourceStatus 代理源状态
type SourceStatus struct {
	ID               int64
	URL              string
	SuccessCount     int
	FailCount        int
	ConsecutiveFails int
	LastSuccess      time.Time
	LastFail         time.Time
	Status           string // active/degraded/disabled
	DisabledUntil    time.Time
}

// LeaseChecker 租约检查接口（由 session.Manager 实现）
type LeaseChecker interface {
	IsLeased(proxyID int64) bool
	GetLeasedIDs() []int64
}

type Storage struct {
	db           *sql.DB
	leaseChecker LeaseChecker
}

const proxySelectColumns = `id, address, protocol, exit_ip, exit_location, country_code, timezone, latency, quality_grade, quality_score, risk_count,
		use_count, success_count, fail_count, last_used, last_check, created_at, status`

// SetLeaseChecker 设置租约检查器（启动时由 main.go 注入）
func (s *Storage) SetLeaseChecker(lc LeaseChecker) {
	s.leaseChecker = lc
}

func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite 单写

	s := &Storage{db: db}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Storage) initSchema() error {
	// 创建代理表
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS proxies (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			address        TEXT NOT NULL UNIQUE,
			protocol       TEXT NOT NULL,
			exit_ip        TEXT NOT NULL DEFAULT '',
			exit_location  TEXT NOT NULL DEFAULT '',
			latency        INTEGER NOT NULL DEFAULT 0,
			quality_grade  TEXT NOT NULL DEFAULT 'C',
			quality_score  INTEGER NOT NULL DEFAULT 0,
			risk_count     INTEGER NOT NULL DEFAULT 0,
			use_count      INTEGER NOT NULL DEFAULT 0,
			success_count  INTEGER NOT NULL DEFAULT 0,
			fail_count     INTEGER NOT NULL DEFAULT 0,
			last_used      DATETIME,
			last_check     DATETIME,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status         TEXT NOT NULL DEFAULT 'active'
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_protocol_latency ON proxies(protocol, latency)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_quality_grade ON proxies(quality_grade, latency)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_quality_score ON proxies(quality_score, latency)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_status ON proxies(status)`)

	// 创建源状态表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS source_status (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			url               TEXT NOT NULL UNIQUE,
			success_count     INTEGER NOT NULL DEFAULT 0,
			fail_count        INTEGER NOT NULL DEFAULT 0,
			consecutive_fails INTEGER NOT NULL DEFAULT 0,
			last_success      DATETIME,
			last_fail         DATETIME,
			status            TEXT NOT NULL DEFAULT 'active',
			disabled_until    DATETIME
		)
	`)
	if err != nil {
		return err
	}

	// 迁移：处理旧的 location 字段（如果存在）
	var hasOldLocation int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='location'`).Scan(&hasOldLocation)
	if err == nil && hasOldLocation > 0 {
		log.Println("[storage] migrating: renaming location to exit_location")
		// 如果有旧的 location 字段，先添加新字段再复制数据
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_location TEXT NOT NULL DEFAULT ''`)
		s.db.Exec(`UPDATE proxies SET exit_location = location WHERE location != ''`)
	}

	// 迁移：添加 exit_ip 字段
	var hasExitIP int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='exit_ip'`).Scan(&hasExitIP)
	if err == nil && hasExitIP == 0 {
		log.Println("[storage] migrating: adding exit_ip column")
		_, err = s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_ip TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("migrate exit_ip column: %w", err)
		}
	}

	// 迁移：添加 exit_location 字段
	var hasExitLocation int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='exit_location'`).Scan(&hasExitLocation)
	if err == nil && hasExitLocation == 0 {
		log.Println("[storage] migrating: adding exit_location column")
		_, err = s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_location TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("migrate exit_location column: %w", err)
		}
	}

	// 迁移：添加 latency 字段
	var hasLatency int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='latency'`).Scan(&hasLatency)
	if err == nil && hasLatency == 0 {
		log.Println("[storage] migrating: adding latency column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN latency INTEGER NOT NULL DEFAULT 0`)
	}

	// 迁移：添加质量等级字段
	var hasQuality int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='quality_grade'`).Scan(&hasQuality)
	if hasQuality == 0 {
		log.Println("[storage] migrating: adding quality_grade column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN quality_grade TEXT NOT NULL DEFAULT 'C'`)
	}

	// 迁移：添加综合质量分字段
	var hasQualityScore int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='quality_score'`).Scan(&hasQualityScore)
	if hasQualityScore == 0 {
		log.Println("[storage] migrating: adding quality_score column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN quality_score INTEGER NOT NULL DEFAULT 0`)
	}

	// 迁移：添加风险计数字段
	var hasRiskCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='risk_count'`).Scan(&hasRiskCount)
	if hasRiskCount == 0 {
		log.Println("[storage] migrating: adding risk_count column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN risk_count INTEGER NOT NULL DEFAULT 0`)
	}

	// 迁移：添加使用统计字段
	var hasUseCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='use_count'`).Scan(&hasUseCount)
	if hasUseCount == 0 {
		log.Println("[storage] migrating: adding usage tracking columns")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN use_count INTEGER NOT NULL DEFAULT 0`)
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN success_count INTEGER NOT NULL DEFAULT 0`)
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN last_used DATETIME`)
	}

	// 迁移：添加状态字段
	var hasStatus int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='status'`).Scan(&hasStatus)
	if hasStatus == 0 {
		log.Println("[storage] migrating: adding status column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`)
	}

	// 迁移：添加地理信息字段
	var hasCountryCode int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='country_code'`).Scan(&hasCountryCode)
	if hasCountryCode == 0 {
		log.Println("[storage] migrating: adding country_code and timezone columns")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN country_code TEXT NOT NULL DEFAULT ''`)
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN timezone TEXT NOT NULL DEFAULT ''`)
	}

	// 创建 sessions 表（会话租约）
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id              TEXT PRIMARY KEY,
			task_id         TEXT NOT NULL,
			proxy_id        INTEGER NOT NULL,
			proxy_address   TEXT NOT NULL,
			protocol        TEXT NOT NULL DEFAULT 'socks5',
			state           TEXT NOT NULL DEFAULT 'active',
			version         INTEGER NOT NULL DEFAULT 1,
			leased_at       INTEGER NOT NULL,
			expires_at      INTEGER NOT NULL,
			released_at     INTEGER,
			result          TEXT,
			risk_detected   INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	// 迁移：修复 sessions task_id 唯一索引（排除空 task_id）
	s.db.Exec(`DROP INDEX IF EXISTS idx_sessions_task_active`)
	s.db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_task_active ON sessions(task_id) WHERE state = 'active' AND task_id != ''`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_proxy_id ON sessions(proxy_id)`)

	// 创建代理日使用量表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS proxy_usage_daily (
			proxy_id    INTEGER NOT NULL,
			usage_date  TEXT NOT NULL,
			use_count   INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (proxy_id, usage_date)
		)
	`)
	if err != nil {
		return err
	}

	return s.backfillQualityScores()
}

// AddProxy 新增代理，已存在则忽略
func (s *Storage) AddProxy(address, protocol string) error {
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO proxies (address, protocol) VALUES (?, ?)`,
		address, protocol,
	)
	if err != nil {
		log.Printf("[storage] AddProxy %s error: %v", address, err)
		return err
	}

	// 检查是否真的插入了
	affected, _ := result.RowsAffected()
	if affected == 0 {
		log.Printf("[storage] AddProxy %s ignored (already exists or constraint)", address)
	}
	return nil
}

// AddProxies 批量新增
func (s *Storage) AddProxies(proxies []Proxy) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO proxies (address, protocol) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, p := range proxies {
		if _, err := stmt.Exec(p.Address, p.Protocol); err != nil {
			log.Printf("insert proxy %s error: %v", p.Address, err)
		}
	}
	return tx.Commit()
}

// getLeasedSet 获取当前被租用的代理 ID 集合
func (s *Storage) getLeasedSet() map[int64]bool {
	if s.leaseChecker == nil {
		return nil
	}
	ids := s.leaseChecker.GetLeasedIDs()
	if len(ids) == 0 {
		return nil
	}
	set := make(map[int64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}

// GetRandom 随机取一个可用代理（优先选择质量高的，排除被租用的）
func (s *Storage) GetRandom() (*Proxy, error) {
	leased := s.getLeasedSet()

	// 优先从 S/A 级代理中随机选择
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s
		 FROM proxies
		 WHERE status = 'active' AND fail_count < 3
		 ORDER BY quality_score DESC, latency ASC, RANDOM()
		 LIMIT 10`, proxySelectColumns),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		if leased != nil && leased[p.ID] {
			continue
		}
		return p, nil
	}
	return nil, fmt.Errorf("no available proxy")
}

type proxyScanner interface {
	Scan(dest ...interface{}) error
}

// scanProxy 扫描代理行数据
func scanProxy(scanner proxyScanner) (*Proxy, error) {
	p := &Proxy{}
	var lastUsed, lastCheck sql.NullTime
	if err := scanner.Scan(&p.ID, &p.Address, &p.Protocol, &p.ExitIP, &p.ExitLocation,
		&p.CountryCode, &p.Timezone,
		&p.Latency, &p.QualityGrade, &p.QualityScore, &p.RiskCount,
		&p.UseCount, &p.SuccessCount, &p.FailCount,
		&lastUsed, &lastCheck, &p.CreatedAt, &p.Status); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		p.LastUsed = lastUsed.Time
	}
	if lastCheck.Valid {
		p.LastCheck = lastCheck.Time
	}
	return p, nil
}

// GetAll 获取所有可用代理（排除被租用的）
func (s *Storage) GetAll() ([]Proxy, error) {
	leased := s.getLeasedSet()

	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3
		 ORDER BY quality_score DESC, latency ASC`, proxySelectColumns),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		if leased != nil && leased[p.ID] {
			continue
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// GetRandomExclude 排除指定地址随机取一个
func (s *Storage) GetRandomExclude(excludes []string) (*Proxy, error) {
	proxies, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	excludeMap := make(map[string]bool)
	for _, e := range excludes {
		excludeMap[e] = true
	}

	var available []Proxy
	for _, p := range proxies {
		if !excludeMap[p.Address] {
			available = append(available, p)
		}
	}

	if len(available) == 0 {
		// 没有可排除的了，随机取任意一个
		return s.GetRandom()
	}

	p := available[rand.Intn(len(available))]
	return &p, nil
}

// GetLowestLatencyExclude 排除指定地址后获取延迟最低的代理
func (s *Storage) GetLowestLatencyExclude(excludes []string) (*Proxy, error) {
	proxies, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	excludeMap := make(map[string]bool)
	for _, e := range excludes {
		excludeMap[e] = true
	}

	var selected *Proxy
	for _, p := range proxies {
		if !excludeMap[p.Address] {
			if selected == nil || p.Latency < selected.Latency {
				proxy := p
				selected = &proxy
			}
		}
	}

	if selected != nil {
		return selected, nil
	}

	return nil, fmt.Errorf("no available proxy")
}

// GetRandomByProtocolExclude 按协议获取随机代理（排除已尝试的）
func (s *Storage) GetRandomByProtocolExclude(protocol string, excludes []string) (*Proxy, error) {
	proxies, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	excludeMap := make(map[string]bool)
	for _, e := range excludes {
		excludeMap[e] = true
	}

	var available []Proxy
	for _, p := range proxies {
		if p.Protocol == protocol && !excludeMap[p.Address] {
			available = append(available, p)
		}
	}

	if len(available) == 0 {
		return nil, fmt.Errorf("no %s proxy available", protocol)
	}

	proxy := available[time.Now().UnixNano()%int64(len(available))]
	return &proxy, nil
}

// GetLowestLatencyByProtocolExclude 按协议获取最低延迟代理（排除已尝试的）
func (s *Storage) GetLowestLatencyByProtocolExclude(protocol string, excludes []string) (*Proxy, error) {
	proxies, err := s.GetAll()
	if err != nil {
		return nil, err
	}

	excludeMap := make(map[string]bool)
	for _, e := range excludes {
		excludeMap[e] = true
	}

	var selected *Proxy
	for _, p := range proxies {
		if p.Protocol == protocol && !excludeMap[p.Address] {
			if selected == nil || p.Latency < selected.Latency {
				proxy := p
				selected = &proxy
			}
		}
	}

	if selected != nil {
		return selected, nil
	}

	return nil, fmt.Errorf("no %s proxy available", protocol)
}

// Delete 立即删除指定代理
func (s *Storage) Delete(address string) error {
	// 保护被会话租用的代理
	if s.leaseChecker != nil {
		var id int64
		s.db.QueryRow(`SELECT id FROM proxies WHERE address = ?`, address).Scan(&id)
		if id > 0 && s.leaseChecker.IsLeased(id) {
			log.Printf("[storage] 跳过删除: %s (id=%d) 正在被会话租用", address, id)
			return nil
		}
	}
	_, err := s.db.Exec(`DELETE FROM proxies WHERE address = ?`, address)
	return err
}

// IncrFail 增加失败次数
func (s *Storage) IncrFail(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = fail_count + 1, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// ResetFail 重置失败次数（验证通过）
func (s *Storage) ResetFail(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = 0, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// UpdateLatency 更新代理的延迟信息（毫秒）
func (s *Storage) UpdateLatency(address string, latencyMs int) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET latency = ?, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		latencyMs, address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// UpdateExitInfo 更新代理的出口 IP、位置、地理信息和综合质量
func (s *Storage) UpdateExitInfo(address, exitIP, exitLocation string, latencyMs int, countryCode, timezone string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET exit_ip = ?, exit_location = ?, country_code = ?, timezone = ?, latency = ?, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		exitIP, exitLocation, countryCode, timezone, latencyMs, address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// RecordProxyUse 记录代理使用（成功）
func (s *Storage) RecordProxyUse(address string, success bool) error {
	if success {
		_, err := s.db.Exec(
			`UPDATE proxies SET use_count = use_count + 1, success_count = success_count + 1, 
			 last_used = CURRENT_TIMESTAMP WHERE address = ?`,
			address,
		)
		if err != nil {
			return err
		}
		return s.recalculateQualityByAddress(address)
	}
	_, err := s.db.Exec(
		`UPDATE proxies SET use_count = use_count + 1, fail_count = fail_count + 1, 
		 last_used = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// GetWorstProxies 获取指定协议中延迟最高的N个代理
func (s *Storage) GetWorstProxies(protocol string, limit int) ([]Proxy, error) {
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s
		 FROM proxies 
		 WHERE protocol = ? AND status = 'active' 
		   AND quality_grade != 'S'
		   AND (JULIANDAY('now') - JULIANDAY(created_at)) * 1440 > 60
		 ORDER BY quality_score ASC, latency DESC, fail_count DESC
		 LIMIT ?`, proxySelectColumns), protocol, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// ReplaceProxy 替换代理（删除旧的，添加新的）
func (s *Storage) ReplaceProxy(oldAddress string, newProxy Proxy) error {
	// 保护被会话租用的代理
	if s.leaseChecker != nil {
		var id int64
		s.db.QueryRow(`SELECT id FROM proxies WHERE address = ?`, oldAddress).Scan(&id)
		if id > 0 && s.leaseChecker.IsLeased(id) {
			log.Printf("[storage] 跳过替换: %s (id=%d) 正在被会话租用", oldAddress, id)
			return fmt.Errorf("proxy is leased")
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧代理
	_, err = tx.Exec(`DELETE FROM proxies WHERE address = ?`, oldAddress)
	if err != nil {
		return err
	}

	// 添加新代理（带完整信息）
	score, grade := CalculateQualitySnapshot(newProxy)
	_, err = tx.Exec(
		`INSERT INTO proxies (address, protocol, exit_ip, exit_location, country_code, timezone, latency, quality_grade, quality_score, risk_count, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active')`,
		newProxy.Address, newProxy.Protocol, newProxy.ExitIP, newProxy.ExitLocation, newProxy.CountryCode, newProxy.Timezone, newProxy.Latency, grade, score, newProxy.RiskCount,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// MarkAsReplacementCandidate 标记代理为替换候选
func (s *Storage) MarkAsReplacementCandidate(addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}
	placeholders := make([]string, len(addresses))
	args := make([]interface{}, len(addresses))
	for i, addr := range addresses {
		placeholders[i] = "?"
		args[i] = addr
	}
	query := fmt.Sprintf(`UPDATE proxies SET status = 'candidate_replace' WHERE address IN (%s)`,
		fmt.Sprintf("%s", placeholders))
	_, err := s.db.Exec(query, args...)
	return err
}

// GetAverageLatency 获取指定协议的平均延迟
func (s *Storage) GetAverageLatency(protocol string) (int, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(
		`SELECT AVG(latency) FROM proxies WHERE protocol = ? AND status = 'active' AND latency > 0`,
		protocol,
	).Scan(&avg)
	if err != nil || !avg.Valid {
		return 0, err
	}
	return int(avg.Float64), nil
}

// GetQualityDistribution 获取质量分布统计
func (s *Storage) GetQualityDistribution() (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT quality_grade, COUNT(*) as count 
		 FROM proxies 
		 WHERE status = 'active' AND fail_count < 3
		 GROUP BY quality_grade`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int)
	for rows.Next() {
		var grade string
		var count int
		if err := rows.Scan(&grade, &count); err != nil {
			return nil, err
		}
		dist[grade] = count
	}
	return dist, nil
}

// GetBatchForHealthCheck 获取一批需要健康检查的代理（排除被租用的）
func (s *Storage) GetBatchForHealthCheck(batchSize int, skipSGrade bool) ([]Proxy, error) {
	leased := s.getLeasedSet()

	// 多查一些以补偿过滤掉的租用代理
	fetchSize := batchSize
	if leased != nil {
		fetchSize += len(leased)
	}

	query := `SELECT id, address, protocol, exit_ip, exit_location, country_code, timezone, latency, quality_grade,
		        quality_score, risk_count, use_count, success_count, fail_count, last_used, last_check, created_at, status
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3`

	if skipSGrade {
		query += ` AND quality_grade != 'S'`
	}

	query += ` ORDER BY
		COALESCE(last_check, '1970-01-01') ASC,
		quality_score DESC,
		latency ASC
		LIMIT ?`

	rows, err := s.db.Query(query, fetchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		if leased != nil && leased[p.ID] {
			continue
		}
		proxies = append(proxies, *p)
		if len(proxies) >= batchSize {
			break
		}
	}
	return proxies, nil
}

// DeleteInvalid 删除失败次数超过阈值的代理
func (s *Storage) DeleteInvalid(maxFailCount int) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE fail_count >= ?`, maxFailCount)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DeleteBlockedCountries 删除指定国家代码出口的代理
func (s *Storage) DeleteBlockedCountries(countryCodes []string) (int64, error) {
	if len(countryCodes) == 0 {
		return 0, nil
	}

	var totalDeleted int64
	for _, code := range countryCodes {
		// exit_location 格式：如 "CN Beijing" 或 "HK Hong Kong"
		// 使用 LIKE 'CODE %' 来匹配国家代码（后面有空格表示有城市信息）
		res, err := s.db.Exec(`DELETE FROM proxies WHERE exit_location LIKE ?`, code+" %")
		if err != nil {
			return totalDeleted, err
		}
		affected, _ := res.RowsAffected()
		totalDeleted += affected
	}
	return totalDeleted, nil
}

// DeleteWithoutExitInfo 删除没有出口信息的代理
func (s *Storage) DeleteWithoutExitInfo() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE exit_ip = '' OR exit_location = ''`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Count 返回可用代理数量
func (s *Storage) Count() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3`,
	).Scan(&count)
	return count, err
}

// CountByProtocol 按协议统计数量
func (s *Storage) CountByProtocol(protocol string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3 AND protocol = ?`,
		protocol,
	).Scan(&count)
	return count, err
}

// IncrementFailCount 增加失败次数
func (s *Storage) IncrementFailCount(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = fail_count + 1, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err != nil {
		return err
	}
	return s.recalculateQualityByAddress(address)
}

// GetByProtocol 按协议获取代理列表
func (s *Storage) GetByProtocol(protocol string) ([]Proxy, error) {
	rows, err := s.db.Query(
		fmt.Sprintf(`SELECT %s
		 FROM proxies 
		 WHERE status IN ('active', 'degraded') AND fail_count < 3 AND protocol = ?
		 ORDER BY quality_score DESC, latency ASC`, proxySelectColumns), protocol,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// IncrementRiskCount 增加风险计数并重算质量
func (s *Storage) IncrementRiskCount(proxyID int64) (int, string, error) {
	_, err := s.db.Exec(`UPDATE proxies SET risk_count = risk_count + 1 WHERE id = ?`, proxyID)
	if err != nil {
		return 0, "", err
	}
	return s.recalculateQualityByID(proxyID)
}

// RecordProxyOutcome 记录代理使用结果并重算质量
func (s *Storage) RecordProxyOutcome(proxyID int64, address, result string, riskDetected bool) (int, string, error) {
	success := result == "success" && !riskDetected
	if err := s.RecordProxyUse(address, success); err != nil {
		return 0, "", err
	}
	if riskDetected || result == "risk_blocked" {
		return s.IncrementRiskCount(proxyID)
	}
	return s.recalculateQualityByID(proxyID)
}

func (s *Storage) backfillQualityScores() error {
	rows, err := s.db.Query(`SELECT id FROM proxies WHERE quality_score = 0`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		if _, _, err := s.recalculateQualityByID(id); err != nil {
			return err
		}
	}
	return nil
}

func (s *Storage) recalculateQualityByAddress(address string) error {
	_, _, err := s.recalculateQualityByLookup(`SELECT `+proxySelectColumns+` FROM proxies WHERE address = ?`, address)
	if err == sql.ErrNoRows {
		return nil
	}
	return err
}

func (s *Storage) recalculateQualityByID(proxyID int64) (int, string, error) {
	return s.recalculateQualityByLookup(`SELECT `+proxySelectColumns+` FROM proxies WHERE id = ?`, proxyID)
}

func (s *Storage) recalculateQualityByLookup(query string, arg interface{}) (int, string, error) {
	p, err := scanProxy(s.db.QueryRow(query, arg))
	if err != nil {
		return 0, "", err
	}

	score, grade := CalculateQualitySnapshot(*p)
	_, err = s.db.Exec(`UPDATE proxies SET quality_score = ?, quality_grade = ? WHERE id = ?`, score, grade, p.ID)
	if err != nil {
		return 0, "", err
	}
	return score, grade, nil
}

// Close 关闭数据库
func (s *Storage) Close() error {
	return s.db.Close()
}

// GetDB 获取数据库实例（供其他模块使用）
func (s *Storage) GetDB() *sql.DB {
	return s.db
}
