package session

import (
	"encoding/json"
	"log"
	"net/http"

	"goproxy/config"
)

// APIHandler Session REST API 处理器
type APIHandler struct {
	manager *Manager
}

// NewAPIHandler 创建 API 处理器
func NewAPIHandler(manager *Manager) *APIHandler {
	return &APIHandler{manager: manager}
}

// RegisterRoutes 注册 Session API 路由到指定的 mux
func (h *APIHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/session/acquire", h.apiKeyAuth(h.handleAcquire))
	mux.HandleFunc("/api/session/release", h.apiKeyAuth(h.handleRelease))
	mux.HandleFunc("/api/session/rotate", h.apiKeyAuth(h.handleRotate))
	mux.HandleFunc("/api/session/status", h.apiKeyAuth(h.handleStatus))
	mux.HandleFunc("/api/session/pool-stats", h.apiKeyAuth(h.handlePoolStats))
	log.Println("[session] API 路由已注册 (/api/session/*)")
}

// apiKeyAuth X-API-Key 认证中间件
func (h *APIHandler) apiKeyAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.Get()
		if cfg.SessionAPIKey == "" {
			// 未配置 API Key 时拒绝所有请求
			jsonError(w, "session API key not configured", http.StatusServiceUnavailable)
			return
		}

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != cfg.SessionAPIKey {
			jsonError(w, "invalid or missing API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// handleAcquire POST /api/session/acquire - 分配独占代理会话
func (h *APIHandler) handleAcquire(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AcquireRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.manager.Acquire(req)
	if err != nil {
		errMsg := err.Error()
		switch errMsg {
		case "max_concurrent_sessions":
			stats := h.manager.GetStats()
			jsonResp(w, http.StatusTooManyRequests, ErrorResponse{
				Error:      errMsg,
				Message:    "已达最大并发会话数",
				Total:      stats.Total,
				Leased:     stats.Leased,
				Cooling:    stats.Cooling,
				RetryAfter: 60,
			})
		case "no_available_proxy":
			stats := h.manager.GetStats()
			jsonResp(w, http.StatusServiceUnavailable, ErrorResponse{
				Error:      errMsg,
				Message:    "无可用代理（全部被租用、冷却中或不达标）",
				Total:      stats.Total,
				Leased:     stats.Leased,
				Cooling:    stats.Cooling,
				RetryAfter: 300,
			})
		default:
			jsonError(w, errMsg, http.StatusInternalServerError)
		}
		return
	}

	jsonResp(w, http.StatusOK, resp)
}

// handleRelease POST /api/session/release - 释放代理会话
func (h *APIHandler) handleRelease(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReleaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		jsonError(w, "session_id required", http.StatusBadRequest)
		return
	}

	resp, err := h.manager.Release(req.SessionID, req.Result, req.RiskDetected)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResp(w, http.StatusOK, resp)
}

// handleRotate POST /api/session/rotate - 会话内 IP 轮换
func (h *APIHandler) handleRotate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RotateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		jsonError(w, "session_id required", http.StatusBadRequest)
		return
	}

	resp, err := h.manager.Rotate(req.SessionID, req.Reason)
	if err != nil {
		errMsg := err.Error()
		switch errMsg {
		case "session_not_found":
			jsonError(w, "session not found", http.StatusNotFound)
		case "session_not_active":
			jsonError(w, "session not active", http.StatusConflict)
		case "no_available_proxy":
			jsonError(w, "no alternative proxy available", http.StatusServiceUnavailable)
		default:
			jsonError(w, errMsg, http.StatusInternalServerError)
		}
		return
	}

	jsonResp(w, http.StatusOK, resp)
}

// handleStatus GET /api/session/status - 查询会话状态
func (h *APIHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	if sessionID != "" {
		// 查询单个会话
		sessions := h.manager.GetActiveSessions()
		for _, s := range sessions {
			if s.ID == sessionID {
				jsonResp(w, http.StatusOK, s)
				return
			}
		}
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}

	// 返回所有活跃会话
	sessions := h.manager.GetActiveSessions()
	jsonResp(w, http.StatusOK, map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handlePoolStats GET /api/session/pool-stats - 获取会话池统计
func (h *APIHandler) handlePoolStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := h.manager.GetStats()
	jsonResp(w, http.StatusOK, stats)
}

// ==================== 辅助函数 ====================

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, message string, status int) {
	jsonResp(w, status, map[string]string{"error": message})
}
