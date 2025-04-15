package main

// An example Bubble Tea server. This will put an ssh session into alt screen
// and continually print up to date terminal information.

import (
	"crypto/md5"
	_ "embed"
	"encoding/hex"

	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/muesli/termenv"
	"github.com/terminaldotshop/terminal/go/pkg/resource"
	"github.com/terminaldotshop/terminal/go/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/recover"
	gossh "golang.org/x/crypto/ssh"
)

type PasswordState int

const (
	PasswordSkip PasswordState = iota
	PasswordPossible
	PasswordWaiting
	PasswordAccepted
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalChan
		cancel()
	}()
	sshPort := os.Getenv("SSH_PORT")
	httpPort := os.Getenv("HTTP_PORT")
	if sshPort == "" {
		sshPort = "2222"
	}
	if httpPort == "" {
		httpPort = "8000"
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort("0.0.0.0", sshPort)),
		wish.WithHostKeyPEM([]byte(resource.Resource.SSHKey.Private)),
		wish.WithMiddleware(
			recover.Middleware(
				bubbletea.Middleware(teaHandler),
				activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
				logging.Middleware(),
			),
		),
		wish.WithPublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			hash := md5.Sum(key.Marshal())
			fingerprint := hex.EncodeToString(hash[:])
			ctx.SetValue("fingerprint", fingerprint)
			ctx.SetValue("anonymous", false)
			return true
		}),
		wish.WithKeyboardInteractiveAuth(
			func(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
				ctx.SetValue("fingerprint", uuid.NewString())
				ctx.SetValue("anonymous", true)
				return true
			},
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	log.Info("Starting SSH server", "port", sshPort)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Error("Could not start server", "error", err)
			cancel()
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://www.terminal.shop", http.StatusFound)
	})

	// Listen on port 80
	go func() {
		defer cancel()
		err := http.ListenAndServe(":"+httpPort, nil)
		if err != nil {
			log.Fatal("ListenAndServe error:", err)
		}
	}()

	<-ctx.Done()
	s.Shutdown(ctx)
	slog.Info("Shutting down server")
}

type sshOutput struct {
	ssh.Session
	tty *os.File
}

func (s *sshOutput) Write(p []byte) (int, error) {
	return s.Session.Write(p)
}

func (s *sshOutput) Read(p []byte) (int, error) {
	return s.Session.Read(p)
}

func (s *sshOutput) Fd() uintptr {
	return s.tty.Fd()
}

// You can wire any Bubble Tea model up to the middleware with a function that
// handles the incoming ssh.Session. Here we just grab the terminal info and
// pass it to the new model. You can also return tea.ProgramOptions (such as
// tea.WithAltScreen) on a session by session basis.
func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	sessionBridge := &sshOutput{
		Session: s,
		tty:     pty.Slave,
	}
	renderer := bubbletea.MakeRenderer(sessionBridge)
	fingerprint := s.Context().Value("fingerprint").(string)
	anonymous := s.Context().Value("anonymous").(bool)
	command := s.Command()
	slog.Info("got fingerprint", "fingerprint", fingerprint)
	slog.Info("got command", "command", command)

	// Get client IP address from the SSH session
	clientAddr := s.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(clientAddr)
	slog.Info("client connected", "ip", host)

	if pty.Term == "xterm-ghostty" {
		renderer.SetColorProfile(termenv.TrueColor)
	}

	model, err := tui.NewModel(renderer, fingerprint, anonymous, &host, command)
	if err != nil {
		return nil, []tea.ProgramOption{}
	}
	return model, []tea.ProgramOption{tea.WithAltScreen()}
}
