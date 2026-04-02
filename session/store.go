package session

import (
	"database/sql"
	"time"
)

// Store 会话持久化层
type Store struct {
	db *sql.DB
}

// NewStore 创建会话存储
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// SaveSession 保存会话到数据库
func (s *Store) SaveSession(sess *Session) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sessions (id, task_id, proxy_id, proxy_address, protocol, state, version, leased_at, expires_at, released_at, result, risk_detected)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.TaskID, sess.ProxyID, sess.ProxyAddress, sess.Protocol,
		sess.State, sess.Version, sess.LeasedAt, sess.ExpiresAt, sess.ReleasedAt,
		"", 0,
	)
	return err
}

// UpdateSessionState 更新会话状态
func (s *Store) UpdateSessionState(sessionID, state string, releasedAt int64, result string, riskDetected bool) error {
	risk := 0
	if riskDetected {
		risk = 1
	}
	_, err := s.db.Exec(`
		UPDATE sessions SET state = ?, version = version + 1, released_at = ?, result = ?, risk_detected = ?
		WHERE id = ? AND state = 'active'`,
		state, releasedAt, result, risk, sessionID,
	)
	return err
}

// IncrDailyUsage 增加代理日使用量
func (s *Store) IncrDailyUsage(proxyID int64) (int, error) {
	today := time.Now().UTC().Format("2006-01-02")
	_, err := s.db.Exec(`
		INSERT INTO proxy_usage_daily (proxy_id, usage_date, use_count) VALUES (?, ?, 1)
		ON CONFLICT(proxy_id, usage_date) DO UPDATE SET use_count = use_count + 1`,
		proxyID, today,
	)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.db.QueryRow(`SELECT use_count FROM proxy_usage_daily WHERE proxy_id = ? AND usage_date = ?`,
		proxyID, today).Scan(&count)
	return count, err
}

// GetDailyUsage 获取代理今日使用量
func (s *Store) GetDailyUsage(proxyID int64) int {
	today := time.Now().UTC().Format("2006-01-02")
	var count int
	s.db.QueryRow(`SELECT COALESCE(use_count, 0) FROM proxy_usage_daily WHERE proxy_id = ? AND usage_date = ?`,
		proxyID, today).Scan(&count)
	return count
}

// GetLastReleasedAt 获取代理上次释放时间
func (s *Store) GetLastReleasedAt(proxyID int64) int64 {
	var releasedAt sql.NullInt64
	s.db.QueryRow(`SELECT MAX(released_at) FROM sessions WHERE proxy_id = ? AND state IN ('released', 'expired')`,
		proxyID).Scan(&releasedAt)
	if releasedAt.Valid {
		return releasedAt.Int64
	}
	return 0
}

// LoadActiveSessions 启动时恢复活跃会话
func (s *Store) LoadActiveSessions() ([]*Session, error) {
	rows, err := s.db.Query(`SELECT id, task_id, proxy_id, proxy_address, protocol, state, version, leased_at, expires_at FROM sessions WHERE state = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		sess := &Session{}
		if err := rows.Scan(&sess.ID, &sess.TaskID, &sess.ProxyID, &sess.ProxyAddress, &sess.Protocol, &sess.State, &sess.Version, &sess.LeasedAt, &sess.ExpiresAt); err != nil {
			continue
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}
