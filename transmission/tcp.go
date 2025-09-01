package transmission

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/schollz/peerdiscovery"
	"github.com/schollz/progressbar/v3"
)

var Debug = 1

const DefaultAutomaticShutdownDelay = 60 * time.Second

type PeerState int8

func (p PeerState) String() string {
	switch p {
	case sender:
		return "sender"
	case receiver:
		return "receiver"
	case relay:
		return "relay"
	case dead:
		return "dead"

	}
	return ""
}

const (
	sender PeerState = iota + 1
	receiver
	relay
	dead
)

type pieceWorker struct {
	index int
	piece [20]byte
}

type Peer struct {
	id      string
	State   PeerState
	Port    int
	portStr string

	mu sync.RWMutex

	OpenFile *VirtualFile
	ZipMode  bool
	//Folder were the zip file will be stored
	ZipFolder         string
	ZipDeleteComplete bool

	Metadata *Metadata

	SenderAddress         string
	ConnectedSenderPort   string
	ConnectedSenderPortV6 string

	//Amount of times we can retry when some piece face a validation error
	//Default is 5
	//When max piece retries is 0 an error is thrown and the transfer is cancelled
	MaxPieceRetries int

	Listeners     []net.Conn
	ListenerLimit int
	Sender        net.Conn

	MulticastAddress string

	DownloadFilePath string

	//Time is seconds that determines how long the server will idle(no listener present) before it closes.
	//Default == 1 minutes
	AutomaticShutdownDelay time.Duration

	//When a new listener joins it's incremented.
	// When a listener completes it decrements. When == 0, notifies the sender to close the file
	ToCommplete int

	shutdown chan struct{}

	bar *progressbar.ProgressBar

	//used internally
	selfConn net.Listener
	wg       sync.WaitGroup
}

type Options struct {
	SenderAddress          string
	FilePath               string
	MaxPieceRetries        int
	ListenerLimit          int
	MulticastAddress       string
	DownloadFilePath       string
	AutomaticShutdownDelay time.Duration
	ZipFolder              string
	ZipDeleteComplete      bool
}

func (p *Peer) broadcast() {

	p.dlog("starting sender server")

	go p.broadcastOnLocalNetwork(false)
	go p.broadcastOnLocalNetwork(true)

}

func (p *Peer) Send(opts Options) error {
	err := p.initSender(opts)
	if err != nil {
		return err
	}
	p.broadcast()

	time.Sleep(500 * time.Millisecond)
	p.run(LOCAL_DEFAULT_ADDRESS)
	return nil
}

func (p *Peer) initSender(opts Options) error {
	id, err := generatePeerID(sender)
	if err != nil {
		return err
	}

	p.id = id

	if p.Port == 0 {
		//Fetch available port
		p.Port = GetFirstOpenPort(LOCAL_DEFAULT_ADDRESS, DEFAULT_PORT)
		p.portStr = fmt.Sprint(p.Port)
	}

	if opts.AutomaticShutdownDelay == 0 {
		opts.AutomaticShutdownDelay = 1 * time.Minute
	}

	p.AutomaticShutdownDelay = opts.AutomaticShutdownDelay

	if opts.ZipFolder != "" {
		opts.FilePath, err = ZipFolder(opts.ZipFolder, opts.FilePath)
		if err != nil {
			return err
		}
	}

	//Generate metadata from file
	meta, vf, err := GenerateMetadata(opts.FilePath)
	if err != nil {
		return err
	}

	p.Metadata = meta
	p.MulticastAddress = opts.MulticastAddress
	p.OpenFile = vf

	p.ZipDeleteComplete = opts.ZipDeleteComplete

	p.ZipFolder = opts.ZipFolder

	if opts.ListenerLimit == 0 {
		opts.ListenerLimit = 4
	}

	p.ListenerLimit = opts.ListenerLimit

	p.shutdown = make(chan struct{})

	return nil
}

func (p *Peer) Listen(opts Options) (err error) {
	p.State = receiver

	if opts.SenderAddress == "" {
		p.dlog("attempting to discover peers")

		var discoveries []peerdiscovery.Discovered
		var wg sync.WaitGroup
		var dmu sync.Mutex

		wg.Add(2)
		go func() {
			defer wg.Done()

			//Ipv4 discoveries
			ipv4Discoveries, err1 := peerdiscovery.Discover(peerdiscovery.Settings{
				Limit:            1,
				Payload:          []byte("ok"),
				TimeLimit:        2 * time.Second,
				Delay:            20 * time.Millisecond,
				MulticastAddress: p.MulticastAddress,
			})

			if err == nil && len(ipv4Discoveries) > 0 {
				dmu.Lock()
				err = err1
				discoveries = append(discoveries, ipv4Discoveries...)
				dmu.Unlock()
			}

		}()
		go func() {
			defer wg.Done()

			//Ipv4 discoveries
			ipv4Discoveries, err1 := peerdiscovery.Discover(peerdiscovery.Settings{
				Limit:            1,
				Payload:          []byte("ok"),
				TimeLimit:        2 * time.Second,
				Delay:            20 * time.Millisecond,
				MulticastAddress: p.MulticastAddress,
				IPVersion:        peerdiscovery.IPv6,
			})

			if err == nil && len(ipv4Discoveries) > 0 {
				dmu.Lock()
				err = err1
				discoveries = append(discoveries, ipv4Discoveries...)
				dmu.Unlock()
			}

		}()
		wg.Wait()

		if err != nil {
			return err
		}

		wasDiscovered := false

		if err == nil && len(discoveries) > 0 {
			p.dlog("all discovered peers %+v\n", discoveries)

			for i, discovered := range discoveries {
				if !bytes.HasPrefix(discovered.Payload, []byte("hello")) {
					p.dlog("skipping discovery %d", i)
					continue
				}

				port := string(bytes.TrimPrefix(discovered.Payload, []byte("hello")))

				if port == "" {
					continue
				}

				address := net.JoinHostPort(discovered.Address, port)
				err := PingServer(address)
				if err == nil {
					p.SenderAddress = address
					wasDiscovered = true
					break
				}
			}
		}

		if !wasDiscovered {
			return fmt.Errorf("no peers found")
		}
	} else {
		p.SenderAddress = opts.SenderAddress
	}

	p.id, _ = generatePeerID(receiver)
	p.MaxPieceRetries = opts.MaxPieceRetries

	conn, err := p.connectToSender()
	if err != nil {
		return err
	}

	if err := p.listenerSenderHandshake(conn); err != nil {
		p.dlog("an error occurred sending sender handshake: %v\n", err)
		conn.Close()
		return err
	}

	if err := p.listenerRequestMetadata(conn); err != nil {
		p.dlog("an error occurred requesting metadata: %v\n", err)
		conn.Close()
		return err
	}

	//Create file from metadata information
	if opts.DownloadFilePath == "" {
		opts.DownloadFilePath = "./"
	}

	// Ensure the download directory exists
	if err := os.MkdirAll(opts.DownloadFilePath, 0755); err != nil {
		return err
	}

	p.DownloadFilePath = opts.DownloadFilePath

	//Build Recevier virtual file from metadata
	p.initializeListenVirtualFile()

	workers := make(chan pieceWorker, len(p.Metadata.Pieces))
	result := make(chan PieceBlock)
	errChan := make(chan error, 1)

	for idx, piece := range p.Metadata.Pieces {
		workers <- pieceWorker{index: idx, piece: piece}
	}

	go func() {
		p.download(workers, conn, result, errChan)
	}()

	p.bar = progressbar.NewOptions64(p.Metadata.FileLength,
		progressbar.OptionSetDescription("Downloading file..."),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(5*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\nDownload completed!\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetPredictTime(true),
	)
	done := 0

	for done < int(p.Metadata.FileLength) {
		select {
		case res := <-result:
			n, err := p.OpenFile.WriteAt(int64(res.Offset), res.Buf)
			if err != nil {
				return nil
			}

			done += n
			p.bar.Add(n)
		case err := <-errChan:
			return err
		}

	}

	_, err = conn.Write(listenerFinishedAck())
	if err != nil {
		return err
	}

	close(workers)
	conn.Close()
	p.OpenFile.Close()
	return nil
}

func (p *Peer) Shutdown() {
	p.mu.Lock()
	if p.State != dead {
		close(p.shutdown)
		p.selfConn.Close()
		p.wg.Wait()
		p.OpenFile.Close()
		p.cleanupZip()
		p.State = dead
	}
	p.mu.Unlock()
}

func (p *Peer) connectToSender() (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", p.SenderAddress, 30*time.Second)
	if err != nil {
		p.dlog("error connecting to sender %v", err)
		return nil, err
	}

	p.dlog("connected to sender on port %s", conn.RemoteAddr().String())

	return conn, err
}

func (p *Peer) download(workers chan pieceWorker, conn net.Conn, result chan PieceBlock, errChan chan error) {
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		p.dlog("an error has occured while listening %v\n", err)
		errChan <- err
	}

	for work := range workers {
		p.dlog("requesting piece %d", work.index)
		_, err := conn.Write(requestPiece(work.index))
		if err != nil {
			p.dlog("an error has occured while listening %v\n", err)
			errChan <- err
		}

		//Expect to read a piece
		msg, err := DeserializeMessageFromReader(conn)
		if err != nil {
			p.dlog("an error has occured while listening %v\n", err)
			errChan <- err
		}
		p.dlog("received piece %d", work.index)

		if msg.ID != MessagePiece {
			p.dlog("message is not a piece")
			errChan <- err
		}

		resPiece, err := UnmarshallPiece(msg)
		if err != nil {
			p.dlog("an error has occured while listening %v\n", err)
			errChan <- err
		}

		if !p.verifyPiece(resPiece) {
			if p.MaxPieceRetries != 0 {
				p.dlog("piece at index %d does not match retrying....", resPiece.Index)
				workers <- work
				p.mu.Lock()
				p.MaxPieceRetries--
				p.mu.Unlock()
				continue
			}

			p.dlog("piece at index %d does not match", resPiece.Index)
			errChan <- fmt.Errorf("piece at index %d does not match", resPiece.Index)
		}

		result <- *resPiece
		conn.SetDeadline(time.Time{})
	}
}

func (p *Peer) run(host string) {
	//Idea: I dont think we need for this logic
	network := "tcp"
	addr := net.JoinHostPort(host, p.portStr)
	if host != "" {
		ip := net.ParseIP(host)
		if ip == nil {
			var tcpIP *net.IPAddr
			tcpIP, err := net.ResolveIPAddr("ip", host)
			if err != nil {
				panic(err)
			}
			ip = tcpIP.IP
		}
		addr = net.JoinHostPort(ip.String(), p.portStr)
		if host != "" {
			if ip.To4() != nil {
				network = "tcp4"
			} else {
				network = "tcp6"
			}
		}
	}

	addr = strings.Replace(addr, "127.0.0.1", "0.0.0.0", 1)
	p.dlog("running sender server on %s", addr)

	l, err := net.Listen(network, addr)
	if err != nil {
		p.dlog(err.Error())
		panic(err)
	}

	fmt.Fprintln(os.Stdout, "Ready to begin sending file")
	fmt.Fprintf(os.Stdout, "Listening on %s\n", l.Addr().String())

	p.mu.Lock()
	p.selfConn = l
	p.mu.Unlock()

	go p.autoShutdown()

	for {
		conn, err := p.selfConn.Accept()
		if err != nil {
			select {
			case <-p.shutdown:
				p.dlog("closing server")
				fmt.Fprintln(os.Stderr, "Closing server....")
				p.Shutdown()
				return
			default:
				p.dlog(err.Error())
				return
			}
		}

		fmt.Fprintf(os.Stdout, "Received connection from %s\n", conn.RemoteAddr().String())
		p.dlog("%s has connected", conn.RemoteAddr().String())

		p.wg.Add(1)
		go func(conn net.Conn) {
			defer func() {
				conn.Close()
				p.wg.Done()
				p.mu.Lock()
				for i, c := range p.Listeners {
					if c == conn {
						p.Listeners = append(p.Listeners[:i], p.Listeners[i+1:]...)
						break
					}
				}
				p.mu.Unlock()

				p.mu.RLock()
				p.dlog("listener %s disconnected, remaining listeners: %d", conn.RemoteAddr(), len(p.Listeners))
				p.mu.RUnlock()
			}()

			for {
				p.dlog("waiting for message from %s", conn.RemoteAddr())
				p.mu.RLock()
				p.dlog("listener length: %d", len(p.Listeners))
				p.mu.RUnlock()
				if err := p.messageProcessor(conn); err != nil {
					p.dlog("listener %s error or EOF: %v", conn.RemoteAddr(), err)
					return
				}
			}
		}(conn)

	}
}

func (p *Peer) autoShutdown() {
	timer := time.NewTimer(p.AutomaticShutdownDelay)

	for {
		<-timer.C

		p.mu.RLock()
		if len(p.Listeners) == 0 {
			p.mu.RUnlock()

			fmt.Fprintln(os.Stdout, "server idled for too long, shutting down....")
			p.dlog("server idled for too long, shutting down....")
			timer.Stop()
			p.Shutdown()
			return
		} else {
			timer.Reset(p.AutomaticShutdownDelay)
		}
		p.mu.RUnlock()
	}

}

func (p *Peer) broadcastOnLocalNetwork(useipv6 bool) {
	p.dlog("broadcasting on local network")
	// look for peers first
	settings := peerdiscovery.Settings{
		Limit:     -1,
		Payload:   []byte("hello" + p.portStr),
		Delay:     20 * time.Millisecond,
		TimeLimit: -1,
	}
	if useipv6 {
		settings.IPVersion = peerdiscovery.IPv6
	} else {
		settings.MulticastAddress = p.MulticastAddress
	}

	discoveries, err := peerdiscovery.Discover(settings)
	p.dlog("discoveries: %+v", discoveries)

	if err != nil {
		p.dlog("an error has occurred while broadcasting", err)
	}
}

// Read an acknowledgement message from the sender
func (p *Peer) listenerSenderHandshake(conn net.Conn) error {
	p.dlog("perform listener sender handshake message")
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}

	defer conn.SetReadDeadline(time.Time{})

	_, err := conn.Write(listenerSenderHandshake())
	if err != nil {
		return err
	}

	msg, err := DeserializeMessageFromReader(conn)
	if err != nil {
		return err
	}

	if msg.ID == MessageListenerAcknowledgement {
		p.Sender = conn
		return nil
	} else {
		p.dlog("panicing sender acknowledgment not received")
		panic("supposed to receive sender acknowledgement")
	}
}

// Read file metadata from sender
func (p *Peer) listenerRequestMetadata(conn net.Conn) error {
	p.dlog("perform request metadata handshake")
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return err
	}

	defer conn.SetReadDeadline(time.Time{})

	_, err := conn.Write(requestMetadata())
	if err != nil {
		return err
	}

	msg, err := DeserializeMessageFromReader(conn)
	if err != nil {
		return err
	}

	if msg.ID == MessageMetadata {
		p.mu.Lock()
		p.Metadata, err = UnmarshallMetadata(msg)
		if err != nil {
			return err
		}

		p.dlog("received metadata from sender")
		p.mu.Unlock()
		return nil
	} else {
		p.dlog("panicing sender acknowledgment not received")
		panic("supposed to receive sender acknowledgement")
	}
}

func (p *Peer) messageProcessor(conn net.Conn) error {
	msg, err := DeserializeMessageFromReader(conn)
	if err != nil {
		return err
	}

	switch msg.ID {
	case MessageListenerSenderHandshake:
		p.dlog("listener detected")

		p.mu.RLock()
		if len(p.Listeners) != p.ListenerLimit {
			p.mu.RUnlock()

			_, err := conn.Write(senderListenerAck())
			if err != nil {
				return err
			}
			p.mu.Lock()
			p.Listeners = append(p.Listeners, conn)
			p.mu.Unlock()
		}

	case MessagePing:
		_, err := conn.Write(sendPong())
		if err != nil {
			return err
		}

	case MessageRequestMetadata:
		p.dlog("%s has requested metadata", conn.RemoteAddr().String())
		msg, err := MarshallMetadata(p.Metadata)
		if err != nil {
			return err
		}
		_, err = conn.Write(msg.Serialize())
		if err != nil {
			return err
		}

	case MessageRequestPiece:
		p.dlog("%s has requested a piece", conn.RemoteAddr().String())

		idx := parsePieceRequest(msg.Payload)
		msg, err := MarshallPiece(p.OpenFile, idx)
		if err != nil {
			return err
		}

		_, err = conn.Write(msg.Serialize())
		if err != nil {
			return err
		}

		p.dlog("sent piece %d to listener: %s", idx, conn.RemoteAddr().String())

	case MessageListenerFinishedAcknowledgement:
		p.dlog("%s has finished downloading", conn.RemoteAddr().String())
		fmt.Fprintf(os.Stdout, "%s has finished downloading\n", conn.RemoteAddr().String())
	}
	return nil
}

// Cleanup zip folder
func (p *Peer) cleanupZip() {
	if p.ZipDeleteComplete {
		_ = os.RemoveAll(p.ZipFolder)
	}
}

func (p *Peer) initializeListenVirtualFile() {
	vf := VirtualFile{
		rootPath:     p.Metadata.Name,
		downloadPath: p.DownloadFilePath,
		files:        p.Metadata.Folders,
		pieces:       p.Metadata.Pieces,
		totalSize:    p.Metadata.FileLength,
		handles:      make([]*os.File, len(p.Metadata.Folders)),
		single:       p.Metadata.Single,
	}
	p.OpenFile = &vf
}

func (p *Peer) verifyPiece(piece *PieceBlock) bool {
	hash := sha1.Sum(piece.Buf)

	return bytes.Equal(hash[:], p.Metadata.Pieces[piece.Index][:])
}

// dlog logs a debugging message if DebugCM > 0.
func (p *Peer) dlog(format string, args ...any) {
	if Debug > 0 {
		format = fmt.Sprintf("[%s] ", p.id) + format
		log.Printf(format, args...)
	}

}

// Generate peer ID from state
func generatePeerID(state PeerState) (string, error) {
	stateStr := state.String()

	byt := make([]byte, 5)
	_, err := rand.Read(byt)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s_%x", stateStr, byt), nil
}

func listenerSenderHandshake() []byte {
	msg := Message{ID: MessageListenerSenderHandshake}
	return msg.Serialize()
}

func senderListenerAck() []byte {
	msg := Message{ID: MessageListenerAcknowledgement}
	return msg.Serialize()
}

func requestMetadata() []byte {
	msg := Message{ID: MessageRequestMetadata}
	return msg.Serialize()
}

func requestPiece(index int) []byte {
	return RequestPiece(index)
}

func listenerFinishedAck() []byte {
	msg := Message{ID: MessageListenerFinishedAcknowledgement}
	return msg.Serialize()
}

func parsePieceRequest(byt []byte) int {
	index := int(binary.BigEndian.Uint32(byt[0:4]))

	return index
}

func sendPong() []byte {
	msg := Message{ID: MessagePong}

	return msg.Serialize()
}

// // Calculate the size of a piece
// func (p *Peer) calculatePieceSize(index int) int {
// 	begin, end := p.calculateBoundsForPiece(index)
// 	return end - begin
// }

// // Given the piece length and file length. Find the boundary of a piece at an index
// func (p *Peer) calculateBoundsForPiece(index int) (begin, end int) {
// 	begin = index * int(p.Metadata.PieceLength)
// 	end = begin + int(p.Metadata.PieceLength)

// 	if end > int(p.Metadata.PieceLength) {
// 		end = int(p.Metadata.FileLength)
// 	}

// 	return begin, end
// }
