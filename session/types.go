package session

import "time"

// Session 会话租约
type Session struct {
	ID           string `json:"session_id"`
	TaskID       string `json:"task_id"`
	ProxyID      int64  `json:"proxy_id"`
	ProxyAddress string `json:"proxy_address"`
	Protocol     string `json:"protocol"`
	ExitIP       string `json:"upstream_ip"`
	CountryCode  string `json:"country_code"`
	Timezone     string `json:"timezone"`
	Grade        string `json:"grade"`
	State        string `json:"state"` // active/releasing/released/expired
	Version      int    `json:"version"`
	LeasedAt     int64  `json:"leased_at"`
	ExpiresAt    int64  `json:"expires_at"`
	ReleasedAt   int64  `json:"released_at,omitempty"`
	UsageToday   int    `json:"usage_today"`
}

// IsExpired 检查会话是否已过期
func (s *Session) IsExpired() bool {
	return time.Now().Unix() > s.ExpiresAt
}

// AcquireRequest 分配请求
type AcquireRequest struct {
	TaskID          string `json:"task_id"`
	TTL             int    `json:"ttl"`
	MinGrade        string `json:"min_grade"`
	CooldownMinutes int    `json:"cooldown_minutes"`
	MaxDailyUses    int    `json:"max_daily_uses"`
	Country         string `json:"country"`
	Protocol        string `json:"protocol"`
}

// AcquireResponse 分配响应
type AcquireResponse struct {
	SessionID      string `json:"session_id"`
	ProxyAddr      string `json:"proxy_addr"`
	UpstreamIP     string `json:"upstream_ip"`
	CountryCode    string `json:"country_code"`
	Timezone       string `json:"timezone"`
	Grade          string `json:"grade"`
	UsageToday     int    `json:"usage_today"`
	LastReleasedAt int64  `json:"last_released_at"`
	ExpiresAt      int64  `json:"expires_at"`
}

// ReleaseRequest 释放请求
type ReleaseRequest struct {
	SessionID    string `json:"session_id"`
	Result       string `json:"result"`        // success/failed/risk_blocked
	RiskDetected bool   `json:"risk_detected"`
}

// ReleaseResponse 释放响应
type ReleaseResponse struct {
	Released     bool   `json:"released"`
	CooldownUntil int64 `json:"cooldown_until"`
	GradeUpdated string `json:"grade_updated"`
}

// RotateRequest 轮换请求
type RotateRequest struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason"`
}

// PoolSessionStats 池子会话统计
type PoolSessionStats struct {
	Total     int `json:"total"`
	Available int `json:"available"`
	Leased    int `json:"leased"`
	Cooling   int `json:"cooling"`
	Exhausted int `json:"exhausted_today"`
}

// ErrorResponse API 错误响应
type ErrorResponse struct {
	Error      string `json:"error"`
	Message    string `json:"message"`
	Total      int    `json:"total,omitempty"`
	Leased     int    `json:"leased,omitempty"`
	Cooling    int    `json:"cooling,omitempty"`
	BelowGrade int   `json:"below_grade,omitempty"`
	RetryAfter int    `json:"retry_after,omitempty"`
}

// gradeRank 评分排序值（越小越好）
func gradeRank(grade string) int {
	switch grade {
	case "S":
		return 1
	case "A":
		return 2
	case "B":
		return 3
	case "C":
		return 4
	default:
		return 5
	}
}

// GradeMeetsMin 检查评分是否达标
func GradeMeetsMin(grade, minGrade string) bool {
	return gradeRank(grade) <= gradeRank(minGrade)
}
