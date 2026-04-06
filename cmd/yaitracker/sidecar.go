package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var sidecarCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "Stdio-to-HTTP MCP proxy with automatic actor management",
	Long: `The sidecar proxies MCP JSON-RPC from stdin/stdout to the central YAITracker
HTTP server. It registers an MCP actor on startup, sends periodic heartbeats,
and revokes the actor on exit. Cursor (or any MCP client) connects to this
command via stdio transport.

Required environment variables:
  YAITRACKER_URL                  Central server URL (e.g. http://localhost:8080)
  YAITRACKER_OAUTH_ACCESS_TOKEN   OAuth bearer token for the human user

Optional environment variables:
  YAITRACKER_OAUTH_REFRESH_TOKEN  Refresh token for automatic renewal (recommended)
  YAITRACKER_REPO_ROOT            Workspace root (auto-detected if omitted)`,
	RunE: runSidecar,
}

func init() {
	rootCmd.AddCommand(sidecarCmd)
}

type sidecarState struct {
	serverURL    string
	httpClient   *http.Client
	mcpSession   string
	mcpSessionMu sync.Mutex

	tokenMu      sync.RWMutex
	accessToken  string
	refreshToken string

	mu             sync.Mutex
	processActorID string
	convActors     map[string]string // conversation_id -> actor_id
	convIDPath     string            // path to .cursor/yait-conversation-id
}

// currentActorID reads the side-channel file for conversation_id and returns
// the per-conversation actor (registering one if needed), or falls back to
// the process-level actor.
func (s *sidecarState) currentActorID() string {
	convID := s.readConversationID()
	if convID == "" {
		return s.processActorID
	}

	s.mu.Lock()
	if id, ok := s.convActors[convID]; ok {
		s.mu.Unlock()
		return id
	}
	s.mu.Unlock()

	label := fmt.Sprintf("sidecar conv=%s pid=%d", convID, os.Getpid())
	id, err := s.registerActor(label)
	if err != nil {
		log.Printf("sidecar: failed to register conversation actor: %v", err)
		return s.processActorID
	}

	s.mu.Lock()
	s.convActors[convID] = id
	s.mu.Unlock()
	log.Printf("sidecar: registered actor for conversation %s", convID[:8])
	return id
}

func (s *sidecarState) currentToken() string {
	s.tokenMu.RLock()
	defer s.tokenMu.RUnlock()
	return s.accessToken
}

func (s *sidecarState) tryRefreshToken() error {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	if s.refreshToken == "" {
		return fmt.Errorf("no refresh token configured")
	}

	body, err := json.Marshal(map[string]string{"refresh_token": s.refreshToken})
	if err != nil {
		return fmt.Errorf("marshal refresh request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/api/v1/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST /api/v1/auth/refresh: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body) //nolint:errcheck // best-effort read
		return fmt.Errorf("refresh failed (status %d): %s", resp.StatusCode, respBody)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode refresh response: %w", err)
	}
	s.accessToken = result.AccessToken
	s.refreshToken = result.RefreshToken
	log.Printf("sidecar: token refreshed successfully")
	return nil
}

func (s *sidecarState) readConversationID() string {
	data, err := os.ReadFile(s.convIDPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// findWorkspaceRoot tries multiple strategies to locate the workspace root:
// 1. Walk up from cwd looking for .git
// 2. Walk up from the executable's directory looking for .git
func findWorkspaceRoot() string {
	candidates := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}
	for _, start := range candidates {
		dir := start
		for {
			if fi, err := os.Stat(filepath.Join(dir, ".git")); err == nil && fi.IsDir() {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

// allActorIDs returns the process actor plus all conversation actors.
func (s *sidecarState) allActorIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, 1+len(s.convActors))
	ids = append(ids, s.processActorID)
	for _, id := range s.convActors {
		ids = append(ids, id)
	}
	return ids
}

func runSidecar(_ *cobra.Command, _ []string) error {
	serverURL := strings.TrimRight(os.Getenv("YAITRACKER_URL"), "/")
	if serverURL == "" {
		return fmt.Errorf("YAITRACKER_URL is required (e.g. http://localhost:8080)")
	}
	accessToken := strings.TrimSpace(os.Getenv("YAITRACKER_OAUTH_ACCESS_TOKEN"))
	if accessToken == "" {
		return fmt.Errorf("YAITRACKER_OAUTH_ACCESS_TOKEN is required")
	}
	refreshToken := strings.TrimSpace(os.Getenv("YAITRACKER_OAUTH_REFRESH_TOKEN"))

	workspaceRoot := os.Getenv("YAITRACKER_REPO_ROOT")
	if workspaceRoot == "" {
		workspaceRoot = findWorkspaceRoot()
	}
	if workspaceRoot == "" {
		return fmt.Errorf("YAITRACKER_REPO_ROOT is required (could not auto-detect workspace root)")
	}
	log.Printf("sidecar: workspace root = %s", workspaceRoot) //nolint:gosec // not user-controlled network input

	state := &sidecarState{
		serverURL:    serverURL,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        5,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		convActors: make(map[string]string),
		convIDPath: workspaceRoot + "/.cursor/yait-conversation-id",
	}

	actorID, err := state.registerActor(fmt.Sprintf("sidecar pid=%d %s", os.Getpid(), time.Now().UTC().Format(time.RFC3339)))
	if err != nil {
		return fmt.Errorf("register actor: %w", err)
	}
	state.processActorID = actorID
	log.Printf("sidecar: registered actor, proxying to %s", serverURL) //nolint:gosec // env var, not tainted network input

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go state.heartbeatLoop(ctx)
	go state.parentWatcher(ctx, cancel)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	err = state.proxyLoop(ctx)
	state.cleanup()
	return err
}

func (s *sidecarState) registerActor(label string) (string, error) {
	body, err := json.Marshal(map[string]string{"label": label})
	if err != nil {
		return "", fmt.Errorf("marshal actor request: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/api/v1/mcp/actors", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.currentToken())
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("POST /api/v1/mcp/actors: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close

	if resp.StatusCode != http.StatusCreated {
		respBody, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return "", fmt.Errorf("POST /api/v1/mcp/actors: status %d (body unreadable)", resp.StatusCode)
		}
		return "", fmt.Errorf("POST /api/v1/mcp/actors: status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode actor response: %w", err)
	}
	return result.ID, nil
}

func (s *sidecarState) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, id := range s.allActorIDs() {
				s.sendHeartbeat(id)
			}
		}
	}
}

func (s *sidecarState) sendHeartbeat(actorID string) {
	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/api/v1/mcp/actors/"+actorID+"/heartbeat", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.currentToken())
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("sidecar: heartbeat failed for %s: %v", actorID, err)
		return
	}
	resp.Body.Close() //nolint:errcheck // best-effort close on heartbeat response
}

func (s *sidecarState) parentWatcher(ctx context.Context, cancel context.CancelFunc) {
	ppid := os.Getppid()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if os.Getppid() != ppid {
				log.Printf("sidecar: parent process changed (was %d), shutting down", ppid)
				cancel()
				return
			}
		}
	}
}

func (s *sidecarState) proxyLoop(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		resp, err := s.forwardRequest(line)
		if err != nil {
			rpcErr := buildJSONRPCError(line, -32603, err.Error())
			fmt.Fprintln(os.Stdout, string(rpcErr)) //nolint:errcheck // stdio write to stdout
			continue
		}
		fmt.Fprintln(os.Stdout, string(resp)) //nolint:errcheck // stdio write to stdout
	}
	return scanner.Err()
}

func (s *sidecarState) forwardRequest(jsonRPC []byte) ([]byte, error) {
	body, statusCode, err := s.doForward(jsonRPC)
	if err != nil {
		return nil, err
	}

	if statusCode == http.StatusUnauthorized && s.refreshToken != "" {
		if refreshErr := s.tryRefreshToken(); refreshErr != nil {
			log.Printf("sidecar: auto-refresh failed: %v", refreshErr)
			return body, nil
		}
		body, _, err = s.doForward(jsonRPC)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

func (s *sidecarState) doForward(jsonRPC []byte) ([]byte, int, error) {
	actorID := s.currentActorID()
	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/mcp", bytes.NewReader(jsonRPC))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+s.currentToken())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if actorID != "" {
		req.Header.Set("X-Yaitracker-Mcp-Actor-Id", actorID)
	}

	s.mcpSessionMu.Lock()
	if s.mcpSession != "" {
		req.Header.Set("Mcp-Session-Id", s.mcpSession)
	}
	s.mcpSessionMu.Unlock()

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("proxy to %s/mcp: %w", s.serverURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close

	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		s.mcpSessionMu.Lock()
		s.mcpSession = sid
		s.mcpSessionMu.Unlock()
	}

	ct := resp.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "text/event-stream") {
		body, err := s.readSSE(resp.Body)
		return body, resp.StatusCode, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}
	return bytes.TrimSpace(body), resp.StatusCode, nil
}

// readSSE reads an SSE stream and returns the last "data:" event payload.
// For MCP Streamable HTTP, the server sends JSON-RPC responses as SSE events.
func (s *sidecarState) readSSE(r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	var lastData []byte
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			lastData = []byte(strings.TrimPrefix(line, "data: "))
		}
	}
	if lastData == nil {
		return nil, fmt.Errorf("no data events in SSE stream")
	}
	return lastData, scanner.Err()
}

func (s *sidecarState) cleanup() {
	for _, id := range s.allActorIDs() {
		s.revokeActor(id)
	}
}

func (s *sidecarState) revokeActor(actorID string) {
	req, err := http.NewRequest(http.MethodDelete, s.serverURL+"/api/v1/mcp/actors/"+actorID, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+s.currentToken())
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("sidecar: failed to revoke actor %s: %v", actorID, err)
		return
	}
	resp.Body.Close() //nolint:errcheck // best-effort close on revoke response
	log.Printf("sidecar: revoked actor %s", actorID)
}

// buildJSONRPCError constructs a JSON-RPC error response matching the request id.
func buildJSONRPCError(reqJSON []byte, code int, message string) []byte {
	var req struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(reqJSON, &req); err != nil {
		req.ID = json.RawMessage("null")
	}
	if req.ID == nil {
		req.ID = json.RawMessage("null")
	}
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      req.ID,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return []byte(`{"jsonrpc":"2.0","id":null,"error":{"code":-32603,"message":"internal error"}}`)
	}
	return b
}
