package transmission

import (
	"net"
	"testing"
	"time"
)

func TestStartAndListen(t *testing.T) {
	// Debug = 0
	p := new(Peer)
	err := p.Start(Options{FilePath: "./testdata/TCP-IP.pdf"})
	if err != nil {
		t.Fatalf("an error as occurred while starting up send %v\n", err)
	}

	l := new(Peer)

	senderAddress := net.JoinHostPort(LOCAL_DEFAULT_ADDRESS, p.portStr)
	err = l.Listen(Options{
		SenderAddress:    senderAddress,
		MaxPieceRetries:  4,
		DownloadFilePath: "./download_test",
	})

	if err != nil {
		t.Fatalf("an error as occurred while listening %v\n", err)
	}
}

func TestStartAndListenConcurrent(t *testing.T) {
	p := new(Peer)
	err := p.Start(Options{FilePath: "./testdata/TCP-IP.pdf"})
	if err != nil {
		t.Fatalf("an error as occurred while starting up send %v\n", err)
	}

	errChan := make(chan error, 2)
	senderAddress := net.JoinHostPort(LOCAL_DEFAULT_ADDRESS, p.portStr)

	go func() {
		l := new(Peer)
		errChan <- l.Listen(Options{
			SenderAddress:    senderAddress,
			MaxPieceRetries:  4,
			DownloadFilePath: "./download_test",
		})

	}()

	go func() {
		l := new(Peer)
		errChan <- l.Listen(Options{
			SenderAddress:    senderAddress,
			MaxPieceRetries:  4,
			DownloadFilePath: "./download_test",
		})

	}()

	time.Sleep(500 * time.Millisecond)
	for range 2 {
		e := <-errChan
		if e != nil {
			t.Fatalf("an error as occurred while listening %v\n", err)
		}
	}

	p.Shutdown()
}

func TestStartAndListenConcurrentDefaultMax(t *testing.T) {
	p := new(Peer)
	err := p.Start(Options{FilePath: "./testdata/TCP-IP.pdf"})
	if err != nil {
		t.Fatalf("an error as occurred while starting up send %v\n", err)
	}

	errChan := make(chan error, p.ListenerLimit)
	senderAddress := net.JoinHostPort(LOCAL_DEFAULT_ADDRESS, p.portStr)

	for range p.ListenerLimit {
		go func() {
			l := new(Peer)
			errChan <- l.Listen(Options{
				SenderAddress:    senderAddress,
				MaxPieceRetries:  4,
				DownloadFilePath: "./download_test",
			})

		}()
	}

	time.Sleep(500 * time.Millisecond)
	for range p.ListenerLimit {
		e := <-errChan
		if e != nil {
			t.Fatalf("an error as occurred while listening %v\n", err)
		}
	}

	p.Shutdown()
}

func TestListenerAutoShutdown(t *testing.T) {
	p := new(Peer)
	err := p.Start(Options{FilePath: "./testdata/TCP-IP.pdf", AutomaticShutdownDelay: 5 * time.Second})
	if err != nil {
		t.Fatalf("an error as occurred while starting up send %v\n", err)
	}

	l := new(Peer)

	senderAddress := net.JoinHostPort(LOCAL_DEFAULT_ADDRESS, p.portStr)
	err = l.Listen(Options{
		SenderAddress:    senderAddress,
		MaxPieceRetries:  4,
		DownloadFilePath: "./download_test",
	})

	if err != nil {
		t.Fatalf("an error as occurred while listening %v\n", err)
	}

	time.Sleep(5 * time.Second)

	if p.State != dead {
		t.Fatalf("expected %s but got %s", dead, p.State)
	}

}
