package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/ss497254/gloski/internal/config"
	"github.com/ss497254/gloski/internal/files"
	"github.com/ss497254/gloski/internal/logger"
	"github.com/ss497254/gloski/internal/system"
	"github.com/ss497254/gloski/proto/agentv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Run starts a stateless device agent. It keeps nothing durable and
// reconnects to the master whenever the stream drops.
func Run(cfg *config.Config, version string) error {
	if cfg.Agent.MasterURL == "" {
		return fmt.Errorf("agent.master_url is required (set GLOSKI_AGENT_MASTER_URL)")
	}
	if cfg.Mode != config.ModeAgent {
		return fmt.Errorf("gloski-agent must run in agent mode (current mode: %s)", cfg.Mode)
	}
	if strings.TrimSpace(cfg.Agent.JoinToken) == "" {
		return fmt.Errorf("agent.join_token is required (set GLOSKI_AGENT_JOIN_TOKEN)")
	}

	name := strings.TrimSpace(cfg.Agent.Name)
	if name == "" {
		name, _ = os.Hostname()
	}
	if name == "" {
		name = "gloski-agent"
	}

	statsStore := system.NewStore(300)
	statsHub := system.NewHub(statsStore)
	collector := system.NewCollector(statsStore, statsHub, 2*time.Second)
	collector.Start()
	defer collector.Stop()
	statsService := system.NewService(statsStore, statsHub)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	backoff := time.Second
	for {
		select {
		case <-quit:
			logger.Info("Agent stopped")
			return nil
		default:
		}

		if err := runSession(cfg, version, name, statsService, quit); err != nil {
			logger.Warn("Agent session ended: %v", err)
		}

		select {
		case <-quit:
			logger.Info("Agent stopped")
			return nil
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func runSession(cfg *config.Config, version, name string, statsService *system.Service, quit <-chan os.Signal) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, err := dialMaster(ctx, cfg.Agent.MasterURL)
	cancel()
	if err != nil {
		return fmt.Errorf("failed to connect to master: %w", err)
	}
	defer conn.Close()

	stream, err := agentv1.NewAgentClient(conn).Connect(context.Background())
	if err != nil {
		return fmt.Errorf("failed to open agent stream: %w", err)
	}

	agent := &agentRuntime{
		cfg:     cfg,
		version: version,
		name:    name,
		stream:  stream,
		stats:   statsService,
		files:   files.NewService(cfg),
		jobs:    make(map[string]context.CancelFunc),
		terms:   make(map[string]*ptySession),
	}
	defer agent.cleanup()
	defer func() { _ = stream.CloseSend() }()

	hostname, _ := os.Hostname()
	stats, _ := statsService.GetStats()
	statsJSON, _ := json.Marshal(stats)

	requestID := strconv.FormatInt(time.Now().UnixNano(), 36)
	if err := agent.send(&agentv1.AgentMessage{
		RequestId: requestID,
		Payload: &agentv1.AgentMessage_Register{
			Register: &agentv1.RegisterRequest{
				JoinToken: cfg.Agent.JoinToken,
				Name:      name,
				Hostname:  hostname,
				Platform:  runtime.GOOS + "/" + runtime.GOARCH,
				Version:   version,
				DeployId:  cfg.Agent.DeployID,
				Master:    cfg.Agent.Master,
				StatsJson: string(statsJSON),
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}

	response, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	registration := response.GetRegisterResponse()
	if registration == nil {
		return errors.New("master did not return a registration response")
	}
	agent.deviceID = registration.GetDeviceId()
	agent.token = registration.GetDeviceToken()
	interval := int(registration.GetHeartbeatInterval())
	if interval <= 0 {
		interval = cfg.Agent.HeartbeatInterval
	}
	if interval <= 0 {
		interval = 10
	}
	logger.Info("Agent %s (%s) registered with master", name, agent.deviceID)

	sessionDone := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(sessionDone) }) }
	go agent.heartbeatLoop(sessionDone, interval, finish)
	go agent.receiveLoop(sessionDone, finish)

	select {
	case <-quit:
		return nil
	case <-sessionDone:
		return errors.New("master stream closed")
	}
}

type agentRuntime struct {
	cfg      *config.Config
	version  string
	name     string
	stream   agentv1.Agent_ConnectClient
	stats    *system.Service
	files    *files.Service
	deviceID string
	token    string

	sendMu sync.Mutex
	mu     sync.Mutex
	jobs   map[string]context.CancelFunc
	terms  map[string]*ptySession
}

func (a *agentRuntime) send(message *agentv1.AgentMessage) error {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()
	return a.stream.Send(message)
}

func (a *agentRuntime) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, cancel := range a.jobs {
		cancel()
		delete(a.jobs, id)
	}
	for id, session := range a.terms {
		session.close()
		delete(a.terms, id)
	}
}

func (a *agentRuntime) heartbeatLoop(done <-chan struct{}, interval int, finish func()) {
	defer finish()
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			stats, err := a.stats.GetStats()
			if err != nil {
				continue
			}
			raw, err := json.Marshal(stats)
			if err != nil {
				continue
			}
			if err := a.send(&agentv1.AgentMessage{
				Payload: &agentv1.AgentMessage_Heartbeat{
					Heartbeat: &agentv1.Heartbeat{StatsJson: string(raw)},
				},
			}); err != nil {
				return
			}
		}
	}
}

func (a *agentRuntime) receiveLoop(done <-chan struct{}, finish func()) {
	defer finish()
	for {
		message, err := a.stream.Recv()
		if err != nil {
			select {
			case <-done:
			default:
				logger.Warn("Agent stream closed: %v", err)
			}
			return
		}
		a.handle(message)
	}
}

func (a *agentRuntime) handle(message *agentv1.MasterMessage) {
	switch payload := message.Payload.(type) {
	case *agentv1.MasterMessage_RunJob:
		go a.runJob(message.RequestId, payload.RunJob)
	case *agentv1.MasterMessage_Cancel:
		a.mu.Lock()
		cancel := a.jobs[message.RequestId]
		a.mu.Unlock()
		if cancel != nil {
			cancel()
		}
	case *agentv1.MasterMessage_FileRequest:
		go a.handleFile(message.RequestId, payload.FileRequest)
	case *agentv1.MasterMessage_ProcessRequest:
		go a.handleProcesses(message.RequestId, payload.ProcessRequest)
	case *agentv1.MasterMessage_TerminalOpen:
		go a.openTerminal(message.RequestId, payload.TerminalOpen)
	case *agentv1.MasterMessage_TerminalInput:
		a.terminalInput(message.RequestId, payload.TerminalInput.GetData())
	case *agentv1.MasterMessage_TerminalClose:
		a.closeTerminal(message.RequestId)
	case *agentv1.MasterMessage_Shutdown:
		logger.Info("Master requested agent shutdown")
		os.Exit(0)
	}
}

func (a *agentRuntime) runJob(id string, request *agentv1.RunJob) {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.jobs[id] = cancel
	a.mu.Unlock()
	defer func() {
		cancel()
		a.mu.Lock()
		delete(a.jobs, id)
		a.mu.Unlock()
	}()

	shell := a.cfg.Shell
	if shell == "" {
		shell = "/bin/sh"
	}
	command := exec.CommandContext(ctx, shell, "-c", request.GetCommand())
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		command.Dir = cwd
	}

	stdout, err := command.StdoutPipe()
	if err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}
	if err := command.Start(); err != nil {
		a.sendResult(id, -1, err.Error())
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		a.streamOutput(id, stdout, false)
	}()
	go func() {
		defer wg.Done()
		a.streamOutput(id, stderr, true)
	}()

	err = command.Wait()
	wg.Wait()
	code := 0
	message := ""
	if err != nil {
		message = err.Error()
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		} else {
			code = -1
		}
	}
	a.sendResult(id, code, message)
}

func (a *agentRuntime) streamOutput(id string, reader interface{ Read([]byte) (int, error) }, stderr bool) {
	buffer := make([]byte, 16*1024)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			sendErr := a.send(&agentv1.AgentMessage{
				RequestId: id,
				Payload: &agentv1.AgentMessage_JobOutput{
					JobOutput: &agentv1.JobOutput{Chunk: chunk, Stderr: stderr},
				},
			})
			if sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (a *agentRuntime) sendResult(id string, code int, message string) {
	_ = a.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload: &agentv1.AgentMessage_JobResult{
			JobResult: &agentv1.JobResult{ExitCode: int32(code), Error: message},
		},
	})
}

func (a *agentRuntime) handleFile(id string, request *agentv1.FileRequest) {
	result := &agentv1.FileResult{}
	switch request.GetOp() {
	case agentv1.FileOp_FILE_OP_LIST:
		listing, err := a.files.List(request.GetPath())
		if err != nil {
			result.Code, result.Message = 1, err.Error()
			break
		}
		raw, err := json.Marshal(listing)
		if err != nil {
			result.Code, result.Message = 1, err.Error()
			break
		}
		result.EntriesJson = string(raw)
	case agentv1.FileOp_FILE_OP_READ:
		content, err := a.files.Read(request.GetPath())
		if err != nil {
			result.Code, result.Message = 1, err.Error()
			break
		}
		result.Content = []byte(content)
	case agentv1.FileOp_FILE_OP_WRITE:
		if err := a.files.Write(request.GetPath(), string(request.GetContent())); err != nil {
			result.Code, result.Message = 1, err.Error()
		}
	case agentv1.FileOp_FILE_OP_DELETE:
		if err := a.files.Delete(request.GetPath()); err != nil {
			result.Code, result.Message = 1, err.Error()
		}
	case agentv1.FileOp_FILE_OP_MKDIR:
		if err := a.files.Mkdir(request.GetPath()); err != nil {
			result.Code, result.Message = 1, err.Error()
		}
	case agentv1.FileOp_FILE_OP_RENAME:
		if err := a.files.Rename(request.GetPath(), request.GetNewPath()); err != nil {
			result.Code, result.Message = 1, err.Error()
		}
	default:
		result.Code, result.Message = 1, "unsupported file operation"
	}
	_ = a.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_FileResult{FileResult: result},
	})
}

func (a *agentRuntime) handleProcesses(id string, request *agentv1.ProcessRequest) {
	limit := int(request.GetLimit())
	if limit <= 0 {
		limit = 20
	}
	processes, err := a.stats.GetProcesses(limit, request.GetSort())
	result := &agentv1.ProcessResult{}
	if err != nil {
		result.Json = jsonLine("error", err.Error())
	} else {
		raw, marshalErr := json.Marshal(processes)
		if marshalErr != nil {
			result.Json = jsonLine("error", marshalErr.Error())
		} else {
			result.Json = string(raw)
		}
	}
	_ = a.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_ProcessResult{ProcessResult: result},
	})
}

func jsonLine(key, value string) string {
	raw, _ := json.Marshal(map[string]string{key: value})
	return string(raw)
}

type ptySession struct {
	ptmx   *os.File
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func (p *ptySession) close() {
	p.cancel()
	_ = p.ptmx.Close()
}

func (a *agentRuntime) openTerminal(id string, request *agentv1.TerminalOpen) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, shell)
	if cwd := strings.TrimSpace(request.GetCwd()); cwd != "" {
		cmd.Dir = cwd
	}
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "TERM=xterm-256color")

	ptmx, err := pty.Start(cmd)
	if err != nil {
		_ = a.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Data: []byte(err.Error() + "\r\n"), Closed: true},
			},
		})
		cancel()
		return
	}
	if request.GetCols() > 0 && request.GetRows() > 0 {
		_ = pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(request.GetRows()), Cols: uint16(request.GetCols())})
	}

	session := &ptySession{ptmx: ptmx, cmd: cmd, cancel: cancel}
	a.mu.Lock()
	a.terms[id] = session
	a.mu.Unlock()
	defer func() {
		session.close()
		a.mu.Lock()
		delete(a.terms, id)
		a.mu.Unlock()
		_, _ = cmd.Process.Wait()
		_ = a.send(&agentv1.AgentMessage{
			RequestId: id,
			Payload: &agentv1.AgentMessage_TerminalOutput{
				TerminalOutput: &agentv1.TerminalOutput{Closed: true},
			},
		})
	}()

	buffer := make([]byte, 16*1024)
	for {
		n, err := ptmx.Read(buffer)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			if sendErr := a.send(&agentv1.AgentMessage{
				RequestId: id,
				Payload: &agentv1.AgentMessage_TerminalOutput{
					TerminalOutput: &agentv1.TerminalOutput{Data: chunk},
				},
			}); sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}

func (a *agentRuntime) terminalInput(id string, data []byte) {
	a.mu.Lock()
	session := a.terms[id]
	a.mu.Unlock()
	if session == nil {
		return
	}
	if len(data) >= 5 && data[0] == 0x01 {
		cols := int(data[1])<<8 | int(data[2])
		rows := int(data[3])<<8 | int(data[4])
		_ = pty.Setsize(session.ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
		return
	}
	_, _ = session.ptmx.Write(data)
}

func (a *agentRuntime) closeTerminal(id string) {
	a.mu.Lock()
	session := a.terms[id]
	a.mu.Unlock()
	if session != nil {
		session.close()
	}
}

func dialMaster(ctx context.Context, raw string) (*grpc.ClientConn, error) {
	raw = strings.TrimSpace(raw)
	transport := insecure.NewCredentials()
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(transport),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32<<20), grpc.MaxCallSendMsgSize(32<<20)),
	}

	target := raw
	if strings.HasPrefix(raw, "unix://") {
		socket := strings.TrimPrefix(raw, "unix://")
		target = "unix://" + socket
		options = append(options, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socket)
		}))
	} else if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		target = parsed.Host
	}
	return grpc.DialContext(ctx, target, options...)
}
