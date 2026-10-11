//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/mrcyjanek/sfos_airpods/internal/aap"
	"github.com/mrcyjanek/sfos_airpods/internal/bluez"
	"github.com/mrcyjanek/sfos_airpods/internal/bus"
)

func main() {
	log.SetFlags(0)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	svc, err := bus.Export(conn)
	if err != nil {
		log.Fatal(err)
	}
	events, err := bluez.Watch(ctx, conn)
	if err != nil {
		log.Fatal(err)
	}

	var (
		wg     sync.WaitGroup
		cancel context.CancelFunc
	)
	stopLink := func() {
		if cancel == nil {
			return
		}
		cancel()
		wg.Wait()
		cancel = nil
	}
	defer stopLink()

	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			stopLink()
			if !ev.Connected {
				svc.MarkDisconnected()
				continue
			}
			var link context.Context
			link, cancel = context.WithCancel(ctx)
			wg.Add(1)
			go func(dev bluez.Device) {
				defer wg.Done()
				runLink(link, dev, svc)
			}(ev.Device)
		}
	}
}

func runLink(ctx context.Context, dev bluez.Device, svc *bus.Service) {
	local, err := bluez.ParseMAC(dev.AdapterAddress)
	if err != nil {
		log.Printf("airpodsd: adapter: %v", err)
		return
	}
	remote, err := bluez.ParseMAC(dev.Address)
	if err != nil {
		log.Printf("airpodsd: device: %v", err)
		return
	}
	for ctx.Err() == nil {
		err := speak(ctx, local, remote, dev, svc)
		if ctx.Err() != nil {
			return
		}
		log.Printf("airpodsd: %v", err)
		svc.MarkDisconnected()
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func speak(ctx context.Context, local, remote [6]byte, dev bluez.Device, svc *bus.Service) error {
	session, err := aap.Dial(ctx, local, remote)
	if err != nil {
		return err
	}
	defer session.Close()
	svc.Attach(session)
	defer svc.Detach(session)
	if err := session.Handshake(ctx, svc.OnPacket); err != nil {
		return err
	}
	svc.MarkConnected(dev.Name, dev.Address)
	log.Printf("airpodsd: control channel open for %s", dev.Name)
	// A socket can connect and then sit forever with no packets. Read used to
	// block until that happened, so battery stayed unknown and noise commands
	// failed. Give the first packet a short window, then drop the channel.
	got := false
	for {
		wait := 5 * time.Second
		if !got {
			wait = 3 * time.Second
		}
		readCtx, cancel := context.WithTimeout(ctx, wait)
		pkt, err := session.Read(readCtx)
		cancel()
		if err == nil {
			got = true
			svc.OnPacket(pkt)
			continue
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) && !got {
			return fmt.Errorf("no control data")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			if err := session.RequestNotifications(ctx); err != nil {
				return err
			}
			continue
		}
		return err
	}
}
