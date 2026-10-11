//go:build linux

package aap

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

const (
	l2capLM        = 0x03
	l2capLMAuth    = 0x0002
	l2capLMEncrypt = 0x0004
	btSecurity     = 4
	btSecurityMed  = 2
)

// Session is one L2CAP control connection. Closing it leaves A2DP alone.
type Session struct {
	mu    sync.Mutex
	fd    int
	Debug bool
	Log   func(string)
}

// Dial opens PSM 0x1001 to remote. local is the adapter address in display
// order; the zero address skips binding. unix.SockaddrL2 reverses the bytes
// into the kernel address layout.
func Dial(ctx context.Context, local, remote [6]byte) (*Session, error) {
	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_SEQPACKET, unix.BTPROTO_L2CAP)
	if err != nil {
		if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
			return nil, fmt.Errorf("permission denied opening L2CAP (retry as root): %w", err)
		}
		return nil, fmt.Errorf("open L2CAP socket: %w", err)
	}
	// Ask for an encrypted channel. The buds are already paired, so this
	// rides the existing ACL and does not touch the audio transport.
	_ = unix.SetsockoptInt(fd, unix.SOL_L2CAP, l2capLM, l2capLMAuth|l2capLMEncrypt)
	_ = unix.SetsockoptInt(fd, unix.SOL_BLUETOOTH, btSecurity, btSecurityMed)
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return nil, err
	}
	if local != ([6]byte{}) {
		if err := unix.Bind(fd, &unix.SockaddrL2{Addr: local, AddrType: unix.BDADDR_BREDR}); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("bind adapter: %w", err)
		}
	}
	remoteAddr := &unix.SockaddrL2{PSM: PSM, Addr: remote, AddrType: unix.BDADDR_BREDR}
	if err := connect(ctx, fd, remoteAddr); err != nil {
		unix.Close(fd)
		if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) {
			return nil, fmt.Errorf("permission denied opening L2CAP (retry as root): %w", err)
		}
		return nil, fmt.Errorf("connect L2CAP: %w", err)
	}
	return &Session{fd: fd}, nil
}

func connect(ctx context.Context, fd int, addr unix.Sockaddr) error {
	err := unix.Connect(fd, addr)
	if err == nil {
		return nil
	}
	if !errors.Is(err, unix.EINPROGRESS) && !errors.Is(err, unix.EAGAIN) {
		return err
	}
	if err := wait(ctx, fd, unix.POLLOUT); err != nil {
		return err
	}
	soerr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
	if err != nil {
		return err
	}
	if soerr != 0 {
		return unix.Errno(soerr)
	}
	return nil
}

func (s *Session) Close() error {
	s.mu.Lock()
	fd := s.fd
	s.fd = -1
	s.mu.Unlock()
	if fd < 0 {
		return nil
	}
	_ = unix.Shutdown(fd, unix.SHUT_RDWR)
	return unix.Close(fd)
}

// Handshake sends the packets the buds require before they report state.
// onPacket receives anything that arrives between the handshake and the
// notification request.
func (s *Session) Handshake(ctx context.Context, onPacket func([]byte)) error {
	if err := s.write(ctx, handshakePacket); err != nil {
		return err
	}
	if err := s.collect(ctx, 250*time.Millisecond, onPacket, nil); err != nil {
		return err
	}
	if err := s.write(ctx, featuresPacket); err != nil {
		return err
	}
	return s.write(ctx, notificationsPacket)
}

// RequestNotifications asks the buds to send battery, noise, and ear state again.
func (s *Session) RequestNotifications(ctx context.Context) error {
	return s.write(ctx, notificationsPacket)
}

// SetNoise writes a listening-mode command.
func (s *Session) SetNoise(mode byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.write(ctx, NoisePacket(mode))
}

// Read returns the next control packet. It blocks until a packet arrives,
// the context is canceled, or the socket closes.
func (s *Session) Read(ctx context.Context) ([]byte, error) {
	return s.read(ctx)
}

func (s *Session) write(ctx context.Context, p []byte) error {
	s.mu.Lock()
	locked := true
	defer func() {
		if locked {
			s.mu.Unlock()
		}
	}()
	fd := s.fd
	if fd < 0 {
		return fmt.Errorf("control channel closed")
	}
	for {
		n, err := unix.Write(fd, p)
		if err == unix.EINTR {
			continue
		}
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			s.mu.Unlock()
			locked = false
			werr := wait(ctx, fd, unix.POLLOUT)
			s.mu.Lock()
			locked = true
			if werr != nil {
				return werr
			}
			if s.fd < 0 {
				return fmt.Errorf("control channel closed")
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("write control packet: %w", err)
		}
		if n != len(p) {
			return fmt.Errorf("short write on control channel (%d/%d)", n, len(p))
		}
		return nil
	}
}

func (s *Session) collect(parent context.Context, d time.Duration, onPacket func([]byte), stop func() bool) error {
	ctx, cancel := context.WithTimeout(parent, d)
	defer cancel()
	for {
		if stop != nil && stop() {
			return nil
		}
		pkt, err := s.read(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
		if s.Debug && s.Log != nil {
			s.Log(hex.EncodeToString(pkt))
		}
		if onPacket != nil {
			onPacket(pkt)
		}
	}
}

func (s *Session) read(ctx context.Context) ([]byte, error) {
	s.mu.Lock()
	fd := s.fd
	s.mu.Unlock()
	if fd < 0 {
		return nil, fmt.Errorf("control channel closed")
	}
	buf := make([]byte, 4096)
	for {
		n, err := unix.Read(fd, buf)
		if err == nil {
			if n == 0 {
				return nil, fmt.Errorf("control channel closed")
			}
			out := make([]byte, n)
			copy(out, buf[:n])
			return out, nil
		}
		if err == unix.EINTR {
			continue
		}
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			if err := wait(ctx, fd, unix.POLLIN); err != nil {
				return nil, err
			}
			continue
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil, context.Canceled
		}
		return nil, err
	}
}

func wait(ctx context.Context, fd int, events int16) error {
	// A timeout must not tear the socket down. Cancel does, so a blocked
	// read returns when the link context ends.
	stop := context.AfterFunc(ctx, func() {
		if errors.Is(ctx.Err(), context.Canceled) {
			_ = unix.Shutdown(fd, unix.SHUT_RDWR)
		}
	})
	defer stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		timeout := -1
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return context.DeadlineExceeded
			}
			timeout = int(remaining.Milliseconds())
			if timeout < 1 {
				timeout = 1
			}
		}
		n, err := unix.Poll([]unix.PollFd{{Fd: int32(fd), Events: events}}, timeout)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			return context.DeadlineExceeded
		}
		return nil
	}
}
