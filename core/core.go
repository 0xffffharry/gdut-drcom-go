package core

import (
	"context"
	"fmt"
	"gdut-drcom-go/lib/auth"
	"gdut-drcom-go/lib/log"
	"golang.org/x/sys/unix"
	"math/rand"
	"net"
	"os"
	"runtime"
	"time"
)

func (d *Drcom) SetLogger(logger *log.Logger) {
	d.logger = logger
}

func (d *Drcom) RunWithContext(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}
	d.ctx = ctx
	if d.logger == nil {
		d.logger = log.NewLogger(os.Stdout, nil)
	}
	return d.runWithContext()
}

func (d *Drcom) runWithContext() error {
	d.logger.Info(d.Tag, "start")
	defer d.logger.Info(d.Tag, "exit")
	if d.RemotePort == 0 {
		d.RemotePort = 61440
	}
	var err error
	if !d.RemoteIP.IsValid() {
		err = fmt.Errorf("invalid remote ip: %s", d.RemoteIP.String())
		d.logger.Fatal(d.Tag, err.Error())
		return err
	}
	var (
		localAddr  = &net.UDPAddr{}
		remoteAddr = &net.UDPAddr{}
	)
	remoteAddr.IP = d.RemoteIP.AsSlice()
	remoteAddr.Port = int(d.RemotePort)
	if d.RemoteIP.Is4() {
		localAddr.IP = net.IPv4zero
	} else if d.RemoteIP.Is6() {
		localAddr.IP = net.IPv6zero
	} else {
		err = fmt.Errorf("invalid remote ip: %s", d.RemoteIP.String())
		d.logger.Fatal(d.Tag, err.Error())
		return err
	}
	localAddr.Port = int(d.RemotePort)
	udpSocket, err := net.DialUDP("udp", localAddr, remoteAddr)
	if err != nil {
		err = fmt.Errorf("listen udp fail: %s", err.Error())
		d.logger.Fatal(d.Tag, err.Error())
		return err
	}
	if d.BindDevice != "" {
		switch runtime.GOOS {
		case "linux":
			f, err := udpSocket.File()
			if err != nil {
				err = fmt.Errorf("get file fail: %s", err.Error())
				d.logger.Fatal(d.Tag, err.Error())
				return err
			}
			err = unix.BindToDevice(int(f.Fd()), d.BindDevice)
			if err != nil {
				err = fmt.Errorf("bind to device fail: %s", err.Error())
				d.logger.Fatal(d.Tag, err.Error())
				return err
			}
			d.logger.Info(d.Tag, fmt.Sprintf("bind to device: %s", d.BindDevice))
		default:
			err = fmt.Errorf("bind device not support")
			d.logger.Fatal(d.Tag, err.Error())
			return err
		}
	}
	switch runtime.GOOS {
	case "linux":
		f, err := udpSocket.File()
		if err != nil {
			err = fmt.Errorf("get file fail: %s", err.Error())
			d.logger.Fatal(d.Tag, err.Error())
			return err
		}
		err = unix.SetsockoptInt(int(f.Fd()), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			err = fmt.Errorf("set socket option fail: %s", err.Error())
			d.logger.Fatal(d.Tag, err.Error())
			return err
		}
	}
	defer func() {
		if udpSocket != nil {
			udpSocket.Close()
		}
	}()
	go func() {
		<-d.ctx.Done()
		udpSocket.Close()
	}()
	for {
		kp1cnt := byte(1)
		retry := 0
		var buf []byte
		pk := auth.MakeKeepAlive1Packet1(kp1cnt)
		for {
			udpSocket.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = udpSocket.Write(pk)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("send keep alive packet 1 => %+v", pk))
			buf = make([]byte, 1024)
			udpSocket.SetReadDeadline(time.Now().Add(readTimeout))
			n, _, err := udpSocket.ReadFromUDP(buf)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("recv keep alive packet 1 <= %+v", buf[:n]))
			break
		}
		if retry == maxRetry {
			err = fmt.Errorf("keep alive fail: %s", err.Error())
			d.logger.Error(d.Tag, err.Error())
			continue
		}
		retry = 0
		kp1cnt++
		kp1cnt %= 0xff
		seed := buf[8:12]
		sip := buf[12:16]
		var data []byte
		if kp1cnt != 1 && kp1cnt != 2 {
			data = auth.MakeKeepAlive1Packet2(seed, sip, d.KeepAlive1Flag, kp1cnt, d.EnableCrypt, false)
		} else {
			data = auth.MakeKeepAlive1Packet2(seed, sip, d.KeepAlive1Flag, kp1cnt, d.EnableCrypt, true)
		}
		for {
			udpSocket.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = udpSocket.Write(data)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("send keep alive packet 2 => %+v", data))
			buf = make([]byte, 1024)
			udpSocket.SetReadDeadline(time.Now().Add(readTimeout))
			n, _, err := udpSocket.ReadFromUDP(buf)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("recv keep alive packet 2 <= %+v", buf[:n]))
			break
		}
		if retry == maxRetry {
			err = fmt.Errorf("keep alive fail: %s", err.Error())
			d.logger.Error(d.Tag, err.Error())
			continue
		}
		select {
		case <-time.After(3 * time.Second):
		case <-d.ctx.Done():
			return nil
		}
		//
		s := rand.NewSource(time.Now().UnixNano())
		n := s.Int63() % 0x10000
		random := []byte{byte(n / 0x100), byte(n % 0x100)}
		kp2cnt := byte(0)
		keepAlive2Key := make([]byte, 4)
		keepAlive2Flag := make([]byte, 2)
		retry = 0
		for {
			data = auth.MakeKeepAlive2Packet1(kp2cnt, keepAlive2Flag, random, keepAlive2Key)
			udpSocket.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = udpSocket.Write(data)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("send keep alive packet 3 => %+v", data))
			buf = make([]byte, 1024)
			udpSocket.SetReadDeadline(time.Now().Add(readTimeout))
			n, _, err := udpSocket.ReadFromUDP(buf)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("recv keep alive packet 3 <= %+v", buf[:n]))
			if buf[0] == 0x07 && buf[2] == 0x10 {
				keepAlive2Flag = buf[6:8]
				kp2cnt++
				continue
			}
			break
		}
		if retry == maxRetry {
			err = fmt.Errorf("keep alive fail: %s", err.Error())
			d.logger.Error(d.Tag, err.Error())
			continue
		}
		keepAlive2Key = buf[16:20]
		kp2cnt++
		retry = 0
		for {
			data = auth.MakeKeepAlive2Packet2(kp2cnt, keepAlive2Flag, random, keepAlive2Key, sip)
			udpSocket.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = udpSocket.Write(data)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("send keep alive packet 4 => %+v", data))
			buf = make([]byte, 1024)
			udpSocket.SetReadDeadline(time.Now().Add(readTimeout))
			n, _, err := udpSocket.ReadFromUDP(buf)
			if err != nil {
				if err == net.ErrClosed {
					return nil
				}
				retry++
				if retry == maxRetry {
					break
				}
				continue
			}
			d.logger.Debug(d.Tag, fmt.Sprintf("recv keep alive packet 4 <= %+v", buf[:n]))
			break
		}
		if retry == maxRetry {
			err = fmt.Errorf("keep alive fail: %s", err.Error())
			d.logger.Error(d.Tag, err.Error())
			continue
		}
		kp2cnt++
		select {
		case <-time.After(17 * time.Second):
		case <-d.ctx.Done():
			return nil
		}
	}
}
