package session

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
	"goproxy/config"
)

// SessionSOCKS5Server Session-Sticky SOCKS5 代理服务器（端口 7781）
// 通过 SOCKS5 用户名/密码认证中的 username 提取 session_id，路由到绑定的上游代理
type SessionSOCKS5Server struct {
	manager *Manager
	port    string
}

// SessionHTTPServer Session-Sticky HTTP 代理服务器（端口 7782）
// 通过 Proxy-Authorization Basic Auth 中的 username 提取 session_id，路由到绑定的上游代理
type SessionHTTPServer struct {
	manager *Manager
	port    string
}

// NewSessionSOCKS5Server 创建 Session-Sticky SOCKS5 服务器
func NewSessionSOCKS5Server(manager *Manager, port string) *SessionSOCKS5Server {
	return &SessionSOCKS5Server{manager: manager, port: port}
}

// NewSessionHTTPServer 创建 Session-Sticky HTTP 服务器
func NewSessionHTTPServer(manager *Manager, port string) *SessionHTTPServer {
	return &SessionHTTPServer{manager: manager, port: port}
}

// ==================== SOCKS5 Session Proxy ====================

// Start 启动 Session-Sticky SOCKS5 服务器
func (s *SessionSOCKS5Server) Start() error {
	log.Printf("[session] SOCKS5 sticky proxy listening on %s", s.port)
	listener, err := net.Listen("tcp", s.port)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection 处理 SOCKS5 连接，从认证中提取 session_id
func (s *SessionSOCKS5Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// SOCKS5 握手（强制要求用户名/密码认证）
	sessionID, err := s.socks5Handshake(clientConn)
	if err != nil {
		log.Printf("[session] socks5 handshake failed: %v", err)
		return
	}

	// 查找会话绑定的上游代理
	sess, err := s.resolveSession(sessionID)
	if err != nil {
		log.Printf("[session] socks5 session resolve failed: session=%s err=%v", sessionID, err)
		return
	}

	// 读取 SOCKS5 请求
	target, err := readSOCKS5Request(clientConn)
	if err != nil {
		log.Printf("[session] socks5 read request failed: %v", err)
		return
	}

	// 通过绑定的上游代理连接目标
	upstreamConn, err := dialViaUpstream(sess.ProxyAddress, sess.Protocol, target)
	if err != nil {
		log.Printf("[session] socks5 dial %s via %s failed: %v (session=%s)",
			target, sess.ProxyAddress, err, sess.ID)
		sendSOCKS5Reply(clientConn, 0x01) // General failure
		return
	}

	// 发送成功响应
	if err := sendSOCKS5Reply(clientConn, 0x00); err != nil {
		upstreamConn.Close()
		return
	}

	log.Printf("[session] socks5 %s via %s (session=%s)", target, sess.ProxyAddress, sess.ID)

	// 双向转发
	go io.Copy(upstreamConn, clientConn)
	io.Copy(clientConn, upstreamConn)
	upstreamConn.Close()
}

// socks5Handshake 处理 SOCKS5 握手，强制用户名/密码认证，验证会话后返回 session_id
func (s *SessionSOCKS5Server) socks5Handshake(conn net.Conn) (string, error) {
	buf := make([]byte, 257)

	// 读取客户端问候: [VER(1), NMETHODS(1), METHODS(1-255)]
	n, err := io.ReadAtLeast(conn, buf, 2)
	if err != nil {
		return "", err
	}

	if buf[0] != 0x05 {
		return "", fmt.Errorf("unsupported SOCKS version: %d", buf[0])
	}

	nmethods := int(buf[1])
	if n < 2+nmethods {
		if _, err := io.ReadFull(conn, buf[n:2+nmethods]); err != nil {
			return "", err
		}
	}

	// 检查客户端是否支持用户名/密码认证 (0x02)
	methods := buf[2 : 2+nmethods]
	supported := false
	for _, m := range methods {
		if m == 0x02 {
			supported = true
			break
		}
	}

	if !supported {
		conn.Write([]byte{0x05, 0xFF}) // No acceptable methods
		return "", fmt.Errorf("client does not support username/password auth")
	}

	// 选择用户名/密码认证
	if _, err := conn.Write([]byte{0x05, 0x02}); err != nil {
		return "", err
	}

	// 读取认证请求: [VER(1), ULEN(1), UNAME(1-255), PLEN(1), PASSWD(1-255)]
	authBuf := make([]byte, 513)
	n, err = io.ReadAtLeast(conn, authBuf, 2)
	if err != nil {
		return "", err
	}

	if authBuf[0] != 0x01 {
		return "", fmt.Errorf("unsupported auth version: %d", authBuf[0])
	}

	ulen := int(authBuf[1])
	if n < 2+ulen {
		if _, err := io.ReadFull(conn, authBuf[n:2+ulen]); err != nil {
			return "", err
		}
		n = 2 + ulen
	}

	sessionID := string(authBuf[2 : 2+ulen])

	// 读取密码长度和密码（密码忽略，只用 username 作为 session_id）
	if n < 2+ulen+1 {
		if _, err := io.ReadFull(conn, authBuf[n:2+ulen+1]); err != nil {
			return "", err
		}
		n = 2 + ulen + 1
	}

	plen := int(authBuf[2+ulen])
	if n < 2+ulen+1+plen {
		if _, err := io.ReadFull(conn, authBuf[n:2+ulen+1+plen]); err != nil {
			return "", err
		}
	}

	// 验证 session_id 格式
	if !strings.HasPrefix(sessionID, "sess-") {
		conn.Write([]byte{0x01, 0x01}) // Auth failed
		return "", fmt.Errorf("invalid session_id format: %s", sessionID)
	}

	// 验证会话是否存在且有效（在发送认证成功之前检查）
	s.manager.mu.Lock()
	sess, ok := s.manager.sessions[sessionID]
	valid := ok && sess.State == "active" && !sess.IsExpired()
	s.manager.mu.Unlock()

	if !valid {
		conn.Write([]byte{0x01, 0x01}) // Auth failed
		return "", fmt.Errorf("session not valid: %s", sessionID)
	}

	// 认证成功
	if _, err := conn.Write([]byte{0x01, 0x00}); err != nil {
		return "", err
	}

	return sessionID, nil
}

// resolveSession 根据 session_id 查找会话和上游代理
func (s *SessionSOCKS5Server) resolveSession(sessionID string) (*Session, error) {
	s.manager.mu.Lock()
	defer s.manager.mu.Unlock()

	sess, ok := s.manager.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session_not_found")
	}
	if sess.State != "active" {
		return nil, fmt.Errorf("session_not_active")
	}
	if sess.IsExpired() {
		return nil, fmt.Errorf("session_expired")
	}

	// 返回副本以在 mutex 外安全使用
	sessCopy := *sess
	return &sessCopy, nil
}

// ==================== HTTP Session Proxy ====================

// Start 启动 Session-Sticky HTTP 服务器
func (s *SessionHTTPServer) Start() error {
	log.Printf("[session] HTTP sticky proxy listening on %s", s.port)
	return http.ListenAndServe(s.port, s)
}

// ServeHTTP 处理 HTTP 代理请求
func (s *SessionHTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 从 Proxy-Authorization 提取 session_id
	sessionID, err := s.extractSessionID(r)
	if err != nil {
		w.Header().Set("Proxy-Authenticate", `Basic realm="GoProxy Session"`)
		http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
		return
	}

	// 查找会话
	sess, err := s.resolveSession(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("session error: %s", err.Error()), http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodConnect {
		s.handleTunnel(w, r, sess)
	} else {
		s.handleHTTP(w, r, sess)
	}
}

// extractSessionID 从 Proxy-Authorization 提取 session_id（作为 username）
func (s *SessionHTTPServer) extractSessionID(r *http.Request) (string, error) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		return "", fmt.Errorf("missing proxy authorization")
	}

	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", fmt.Errorf("invalid auth scheme")
	}

	decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return "", fmt.Errorf("invalid base64")
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return "", fmt.Errorf("invalid credentials format")
	}

	sessionID := credentials[0]
	if !strings.HasPrefix(sessionID, "sess-") {
		return "", fmt.Errorf("invalid session_id format")
	}

	return sessionID, nil
}

// resolveSession 根据 session_id 查找活跃会话
func (s *SessionHTTPServer) resolveSession(sessionID string) (*Session, error) {
	s.manager.mu.Lock()
	defer s.manager.mu.Unlock()

	sess, ok := s.manager.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session_not_found")
	}
	if sess.State != "active" {
		return nil, fmt.Errorf("session_not_active")
	}
	if sess.IsExpired() {
		return nil, fmt.Errorf("session_expired")
	}

	// 返回副本以在 mutex 外使用
	sessCopy := *sess
	return &sessCopy, nil
}

// handleHTTP 处理普通 HTTP 请求
func (s *SessionHTTPServer) handleHTTP(w http.ResponseWriter, r *http.Request, sess *Session) {
	client, err := buildSessionClient(sess)
	if err != nil {
		http.Error(w, "upstream proxy error", http.StatusBadGateway)
		return
	}

	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Del("Proxy-Authorization")
	req.Header.Del("Proxy-Connection")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[session] http %s via %s failed: %v (session=%s)",
			r.RequestURI, sess.ProxyAddress, err, sess.ID)
		http.Error(w, "upstream proxy failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	log.Printf("[session] http %s via %s -> %d (session=%s)",
		r.RequestURI, sess.ProxyAddress, resp.StatusCode, sess.ID)
}

// handleTunnel 处理 HTTPS CONNECT 隧道
func (s *SessionHTTPServer) handleTunnel(w http.ResponseWriter, r *http.Request, sess *Session) {
	upstreamConn, err := dialViaUpstream(sess.ProxyAddress, sess.Protocol, r.Host)
	if err != nil {
		log.Printf("[session] tunnel dial %s via %s failed: %v (session=%s)",
			r.Host, sess.ProxyAddress, err, sess.ID)
		http.Error(w, "upstream proxy failed", http.StatusBadGateway)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		upstreamConn.Close()
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		upstreamConn.Close()
		return
	}

	fmt.Fprintf(clientConn, "HTTP/1.1 200 Connection Established\r\n\r\n")
	log.Printf("[session] tunnel %s via %s established (session=%s)",
		r.Host, sess.ProxyAddress, sess.ID)

	go transfer(upstreamConn, clientConn)
	go transfer(clientConn, upstreamConn)
}

// ==================== 共用辅助函数 ====================

// dialViaUpstream 通过上游代理连接目标
func dialViaUpstream(proxyAddr, protocol, target string) (net.Conn, error) {
	cfg := config.Get()
	timeout := time.Duration(cfg.ValidateTimeout) * time.Second

	switch protocol {
	case "http":
		conn, err := net.DialTimeout("tcp", proxyAddr, timeout)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(conn, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n", target, target)
		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil {
			conn.Close()
			return nil, err
		}
		if n < 12 || string(buf[:12]) != "HTTP/1.1 200" {
			conn.Close()
			return nil, fmt.Errorf("upstream http proxy connect failed")
		}
		return conn, nil

	case "socks5":
		dialer := &net.Dialer{Timeout: timeout}
		proxyConn, err := dialer.Dial("tcp", proxyAddr)
		if err != nil {
			return nil, err
		}

		// SOCKS5 握手（无认证）
		if _, err := proxyConn.Write([]byte{0x05, 0x01, 0x00}); err != nil {
			proxyConn.Close()
			return nil, err
		}

		handshake := make([]byte, 2)
		if _, err := io.ReadFull(proxyConn, handshake); err != nil {
			proxyConn.Close()
			return nil, err
		}
		if handshake[0] != 0x05 || handshake[1] != 0x00 {
			proxyConn.Close()
			return nil, fmt.Errorf("socks5 handshake failed")
		}

		// 发送 CONNECT 请求
		host, port, err := net.SplitHostPort(target)
		if err != nil {
			proxyConn.Close()
			return nil, err
		}

		req := []byte{0x05, 0x01, 0x00}
		if ip := net.ParseIP(host); ip != nil {
			if ip4 := ip.To4(); ip4 != nil {
				req = append(req, 0x01)
				req = append(req, ip4...)
			} else {
				req = append(req, 0x04)
				req = append(req, ip...)
			}
		} else {
			req = append(req, 0x03)
			req = append(req, byte(len(host)))
			req = append(req, []byte(host)...)
		}

		portNum := uint16(0)
		fmt.Sscanf(port, "%d", &portNum)
		portBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(portBytes, portNum)
		req = append(req, portBytes...)

		if _, err := proxyConn.Write(req); err != nil {
			proxyConn.Close()
			return nil, err
		}

		reply := make([]byte, 10)
		if _, err := io.ReadAtLeast(proxyConn, reply, 10); err != nil {
			proxyConn.Close()
			return nil, err
		}
		if reply[1] != 0x00 {
			proxyConn.Close()
			return nil, fmt.Errorf("socks5 connect failed, code: %d", reply[1])
		}

		return proxyConn, nil

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

// buildSessionClient 为会话构建 HTTP 客户端
func buildSessionClient(sess *Session) (*http.Client, error) {
	cfg := config.Get()
	timeout := time.Duration(cfg.ValidateTimeout) * time.Second

	switch sess.Protocol {
	case "http":
		proxyURL, err := url.Parse(fmt.Sprintf("http://%s", sess.ProxyAddress))
		if err != nil {
			return nil, err
		}
		return &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
			Timeout:   timeout,
		}, nil
	case "socks5":
		dialer, err := proxy.SOCKS5("tcp", sess.ProxyAddress, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		return &http.Client{
			Transport: &http.Transport{Dial: dialer.Dial},
			Timeout:   timeout,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", sess.Protocol)
	}
}

// readSOCKS5Request 读取 SOCKS5 连接请求，返回目标地址
func readSOCKS5Request(conn net.Conn) (string, error) {
	buf := make([]byte, 262)

	n, err := io.ReadAtLeast(conn, buf, 4)
	if err != nil {
		return "", err
	}

	if buf[0] != 0x05 {
		return "", fmt.Errorf("invalid version: %d", buf[0])
	}

	cmd := buf[1]
	if cmd != 0x01 {
		sendSOCKS5Reply(conn, 0x07)
		return "", fmt.Errorf("unsupported command: %d", cmd)
	}

	atyp := buf[3]
	var host string
	var addrLen int

	switch atyp {
	case 0x01: // IPv4
		addrLen = 4
		if n < 4+addrLen+2 {
			if _, err := io.ReadFull(conn, buf[n:4+addrLen+2]); err != nil {
				return "", err
			}
		}
		host = fmt.Sprintf("%d.%d.%d.%d", buf[4], buf[5], buf[6], buf[7])
	case 0x03: // Domain
		addrLen = int(buf[4])
		if n < 4+1+addrLen+2 {
			if _, err := io.ReadFull(conn, buf[n:4+1+addrLen+2]); err != nil {
				return "", err
			}
		}
		host = string(buf[5 : 5+addrLen])
	case 0x04: // IPv6
		addrLen = 16
		if n < 4+addrLen+2 {
			if _, err := io.ReadFull(conn, buf[n:4+addrLen+2]); err != nil {
				return "", err
			}
		}
		host = net.IP(buf[4 : 4+addrLen]).String()
	default:
		sendSOCKS5Reply(conn, 0x08)
		return "", fmt.Errorf("unsupported address type: %d", atyp)
	}

	portOffset := 4
	if atyp == 0x03 {
		portOffset = 5 + addrLen
	} else {
		portOffset = 4 + addrLen
	}
	port := binary.BigEndian.Uint16(buf[portOffset : portOffset+2])

	return fmt.Sprintf("%s:%d", host, port), nil
}

// sendSOCKS5Reply 发送 SOCKS5 响应
func sendSOCKS5Reply(conn net.Conn, rep byte) error {
	reply := []byte{
		0x05, rep, 0x00, 0x01,
		0, 0, 0, 0,
		0, 0,
	}
	_, err := conn.Write(reply)
	return err
}

// transfer 双向转发数据
func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}
