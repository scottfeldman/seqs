package stacks_test

import (
	"encoding/hex"
	"errors"
	"log/slog"
	"math"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/soypat/seqs"
	"github.com/soypat/seqs/eth"
	"github.com/soypat/seqs/stacks"
)

const (
	testingLargeNetworkSize = 2 // Minimum=2
	exchangesToEstablish    = 3
	exchangesToClose        = 4
)

func TestDHCP(t *testing.T) {
	const networkSize = testingLargeNetworkSize // How many distinct IP/MAC addresses on network.
	requestedIP := netip.AddrFrom4([4]byte{192, 168, 1, 69})
	siaddr := netip.AddrFrom4([4]byte{192, 168, 1, 1})
	Stacks := createPortStacks(t, networkSize)

	clientStack := Stacks[0]
	serverStack := Stacks[1]

	setLog(clientStack, "cl", slog.LevelDebug)
	setLog(serverStack, "sv", slog.LevelDebug)

	clientStack.SetAddr(netip.AddrFrom4([4]byte{}))
	serverStack.SetAddr(netip.AddrFrom4([4]byte{}))

	client := stacks.NewDHCPClient(clientStack, 68)
	server := stacks.NewDHCPServer(serverStack, siaddr, 67)
	err := client.BeginRequest(stacks.DHCPRequestConfig{
		RequestedAddr: requestedIP,
		Xid:           0x12345678,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = server.Start()
	if err != nil {
		t.Fatal(err)
	}

	const minDHCPSize = eth.SizeEthernetHeader + eth.SizeIPv4Header + eth.SizeUDPHeader + eth.SizeDHCPHeader
	// Client performs DISCOVER.
	egr := NewExchanger(clientStack, serverStack)
	ex, n := egr.DoExchanges(t, 1)
	if n < minDHCPSize {
		t.Errorf("ex[%d] sent=%d want>=%d", ex, n, minDHCPSize)
	}
	if client.Done() {
		t.Fatal("client done on first exchange?!")
	}

	// Server responds with OFFER.
	ex, n = egr.DoExchanges(t, 1)
	if n < minDHCPSize {
		t.Errorf("ex[%d] sent=%d want>=%d", ex, n, minDHCPSize)
	}
	t.Logf("\nclient=%+v\nserver=%+v\n", client, server)
	// Client performs REQUEST.
	ex, n = egr.DoExchanges(t, 1)
	if n < minDHCPSize {
		t.Errorf("ex[%d] sent=%d want>=%d", ex, n, minDHCPSize)
	}
	if client.Done() {
		t.Fatal("client done on request?!")
	}

	// Server performs ACK; client processes ACK
	ex, n = egr.DoExchanges(t, 1)
	if n < minDHCPSize {
		t.Errorf("ex[%d] sent=%d want>=%d", ex, n, minDHCPSize)
	}
	if !client.Done() {
		t.Fatal("client not processed ACK yet")
	}

}

func TestARP(t *testing.T) {
	const networkSize = testingLargeNetworkSize // How many distinct IP/MAC addresses on network.
	stacks := createPortStacks(t, networkSize)

	sender := stacks[0]
	target := stacks[1]
	const expectedARP = eth.SizeEthernetHeader + eth.SizeARPv4Header
	// Send ARP request from sender to target.
	sender.ARP().BeginResolve(target.Addr())
	egr := NewExchanger(stacks...)
	ex, n := egr.DoExchanges(t, 1)
	if n != expectedARP {
		t.Errorf("ex[%d] sent=%d want=%d", ex, n, expectedARP)
	}
	// Target responds to sender.
	ex, n = egr.DoExchanges(t, 1)
	if n != expectedARP {
		t.Errorf("ex[%d] sent=%d want=%d", ex, n, expectedARP)
	}

	ip, mac, err := sender.ARP().ResultAs6()
	if err != nil {
		t.Fatal(err)
	}
	if !ip.IsValid() {
		t.Fatal("invalid IP")
	}
	if mac != target.MACAs6() {
		t.Errorf("result.HardwareSender=%s want=%s", mac, target.MACAs6())
	}
	if ip.As4() != target.Addr().As4() {
		t.Errorf("result.ProtoSender=%s want=%s", ip, target.Addr().As4())
	}

	// No more data to exchange.
	ex, n = egr.DoExchanges(t, 1)
	if n != 0 {
		t.Fatalf("ex[%d] sent=%d want=0", ex, n)
	}
}

func TestTCPEstablish(t *testing.T) {
	client, server := createTCPClientServerPair(t)
	// 3 way handshake needs 3 exchanges to complete.
	const maxTransactions = exchangesToEstablish
	egr := NewExchanger(client.PortStack(), server.PortStack())
	txDone, numBytesSent := egr.DoExchanges(t, maxTransactions)

	_, remnant := egr.DoExchanges(t, 1)
	if remnant != 0 {
		// TODO(soypat): prevent duplicate ACKs from being sent.
		t.Fatalf("duplicate ACK detected? remnant=%d want=0", remnant)
	}

	const expectedData = (eth.SizeEthernetHeader + eth.SizeIPv4Header + eth.SizeTCPHeader) * maxTransactions
	if numBytesSent < expectedData {
		t.Error("too little data exchanged", numBytesSent, " want>=", expectedData)
	}
	if txDone > exchangesToEstablish {
		t.Errorf("too many exchanges for a 3 way handshake: got %d want %d", txDone, exchangesToEstablish)
	} else if txDone < exchangesToEstablish {
		t.Errorf("too few exchanges for a 3 way handshake: got %d want %d", txDone, exchangesToEstablish)
	}
	if client.State() != seqs.StateEstablished {
		t.Errorf("client not established: got %s want %s", client.State(), seqs.StateEstablished)
	}
	if server.State() != seqs.StateEstablished {
		t.Errorf("server not established: got %s want %s", server.State(), seqs.StateEstablished)
	}
}

func TestTCPSendReceive_simplex(t *testing.T) {
	// Create Client+Server and establish TCP connection between them.
	client, server := createTCPClientServerPair(t)
	egr := NewExchanger(client.PortStack(), server.PortStack())
	egr.DoExchanges(t, exchangesToEstablish)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatal("not established")
	}

	// Send data from client to server.
	const data = "hello world"
	socketSendString(client, data)
	egr.DoExchanges(t, 2)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}
	got := socketReadAllString(server)
	if got != data {
		t.Errorf("server: got %q want %q", got, data)
	}
}

func TestTCPSendReceive_duplex_single(t *testing.T) {
	// Create Client+Server and establish TCP connection between them.
	client, server := createTCPClientServerPair(t)
	cstack, sstack := client.PortStack(), server.PortStack()
	egr := NewExchanger(cstack, sstack)
	egr.DoExchanges(t, exchangesToEstablish)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}

	// Send data from client to server.
	const data = "hello world"
	socketSendString(client, data)
	socketSendString(server, data)

	tx, bytes := egr.DoExchanges(t, 2)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}
	t.Logf("tx=%d bytes=%d", tx, bytes)
	clientstr := socketReadAllString(client)
	serverstr := socketReadAllString(server)
	if clientstr != data {
		t.Errorf("client: got %q want %q", clientstr, data)
	}
	if serverstr != data {
		t.Errorf("server: got %q want %q", serverstr, data)
	}
}

func TestTCPSendReceive_duplex(t *testing.T) {
	// Create Client+Server and establish TCP connection between them.
	client, server := createTCPClientServerPair(t)
	egr := NewExchanger(client.PortStack(), server.PortStack())
	egr.DoExchanges(t, exchangesToEstablish)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}

	// Send data from client to server multiple times.
	testSocketDuplex(t, client, server, egr, 1024)
}

func TestTCPClose_noPendingData(t *testing.T) {
	// Create Client+Server and establish TCP connection between them.
	client, server := createTCPClientServerPair(t)
	egr := NewExchanger(client.PortStack(), server.PortStack())
	egr.DoExchanges(t, exchangesToEstablish)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}
	_, b := egr.DoExchanges(t, 2)
	if b != 0 {
		t.Fatal("expected no data to be exchanged after establishment")
	}

	err := client.Close()
	if err != nil {
		t.Fatalf("client.Close(): %v", err)
	}
	i := 0
	doExpect := func(t *testing.T, wantClient, wantServer seqs.State, wantFlags seqs.Flags) {
		t.Helper()
		isRx := i%2 == 0
		if isRx {
			pkts, _ := egr.HandleTx(t)
			if pkts == 0 {
				t.Error("no packet")
			}
			lastSeg := egr.LastSegment()
			if wantFlags != 0 && lastSeg.Flags != wantFlags {
				t.Errorf("do[%d] RX=%v\nwant flags=%v\ngot  flags=%v", i, isRx, wantFlags, lastSeg.Flags)
			}
		} else {
			egr.HandleRx(t)
		}
		t.Logf("client=%s server=%s", client.State(), server.State())
		if client.State() != wantClient || server.State() != wantServer {
			t.Fatalf("do[%d] RX=%v\nwant client=%s server=%s\ngot  client=%s server=%s",
				i, isRx, wantClient, wantServer, client.State(), server.State())
		}
		i++
	}
	// See RFC 9293 Figure 5: TCP Connection State Diagram.
	/*
		Figure 12: Normal Close Sequence
		TCP Peer A                                           TCP Peer B
		1.  ESTABLISHED                                          ESTABLISHED

		2.  (Close)
			FIN-WAIT-1  --> <SEQ=100><ACK=300><CTL=FIN,ACK>  --> CLOSE-WAIT

		3.  FIN-WAIT-2  <-- <SEQ=300><ACK=101><CTL=ACK>      <-- CLOSE-WAIT

		4.                                                       (Close)
			TIME-WAIT   <-- <SEQ=300><ACK=101><CTL=FIN,ACK>  <-- LAST-ACK

		5.  TIME-WAIT   --> <SEQ=101><ACK=301><CTL=ACK>      --> CLOSED

		6.  (2 MSL)
			CLOSED
	*/
	// Peer A == Client;   Peer B == Server
	const finack = seqs.FlagFIN | seqs.FlagACK
	doExpect(t, seqs.StateFinWait1, seqs.StateEstablished, finack)     // do[0] Client sends FIN|ACK
	doExpect(t, seqs.StateFinWait1, seqs.StateCloseWait, 0)            // do[1] Server receives FINACK, goes into close wait
	doExpect(t, seqs.StateFinWait1, seqs.StateCloseWait, seqs.FlagACK) // do[2] Server sends ACK of client FIN
	doExpect(t, seqs.StateFinWait2, seqs.StateCloseWait, 0)            // do[3] client receives ACK of FIN, goes into finwait2
	doExpect(t, seqs.StateFinWait2, seqs.StateLastAck, finack)         // do[4] Server sends out FIN|ACK and enters LastAck state.
	doExpect(t, seqs.StateTimeWait, seqs.StateClosed, 0)               // do[5] Client receives FIN, prepares to send ACK and enters TimeWait state.
	doExpect(t, seqs.StateClosed, seqs.StateClosed, seqs.FlagACK)      // do[6] Client sends ACK and enters Closed state.
}

func TestTCPSocketOpenOfClosedPort(t *testing.T) {
	// Create Client+Server and establish TCP connection between them.
	const newPortoffset = 1
	const newISS = 1337
	client, server := createTCPClientServerPair(t)
	cstack, sstack := client.PortStack(), server.PortStack()

	egr := NewExchanger(cstack, sstack)
	egr.DoExchanges(t, exchangesToEstablish)
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}
	client.Close()
	egr.DoExchanges(t, exchangesToClose)
	if client.State() != seqs.StateClosed || server.State() != seqs.StateClosed {
		t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
	}

	saddrport := netip.AddrPortFrom(sstack.Addr(), server.Port()+newPortoffset)
	err := client.OpenDialTCP(client.Port()+newPortoffset+1, sstack.MACAs6(), saddrport, newISS)
	if err != nil {
		t.Fatal(err)
	}
	err = server.OpenListenTCP(saddrport.Port(), newISS+100)
	if err != nil {
		t.Fatal(err)
	}
	const minBytesToEstablish = (eth.SizeEthernetHeader + eth.SizeIPv4Header + eth.SizeTCPHeader) * exchangesToEstablish
	_, nbytes := egr.DoExchanges(t, exchangesToEstablish)
	if nbytes < minBytesToEstablish {
		t.Fatalf("insufficient data to establish: got %d want>=%d", nbytes, minBytesToEstablish)
	}
	testSocketDuplex(t, client, server, egr, 128)
}

func testSocketDuplex(t *testing.T, client, server *stacks.TCPSocket, egr *Exchanger, messages int) {
	if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
		panic("not established")
	}
	// Send data from client to server multiple times.
	for i := 0; i < messages; i++ {
		istr := strconv.Itoa(i)
		cdata := "hello server " + istr
		sdata := "hello client " + istr

		socketSendString(client, cdata)
		socketSendString(server, sdata)
		tx, bytes := egr.DoExchanges(t, 2)
		if client.State() != seqs.StateEstablished || server.State() != seqs.StateEstablished {
			t.Fatalf("not established: client=%s server=%s", client.State(), server.State())
		}
		_, _ = tx, bytes
		// t.Logf("tx=%d bytes=%d", tx, bytes)
		clientstr := socketReadAllString(client)
		serverstr := socketReadAllString(server)
		if clientstr != sdata {
			t.Errorf("client: got %q want %q", clientstr, sdata)
		}
		if serverstr != cdata {
			t.Errorf("server: got %q want %q", serverstr, cdata)
		}
	}
}

func TestPortStackTCPDecoding(t *testing.T) {
	const dataport = 1234
	packets := []string{
		"28cdc1054d3ed85ed34303eb08004500003c76eb400040063f76c0a80192c0a80178ee1604d2a0ceb98a00000000a002faf06e800000020405b40402080a14ccf8250000000001030307",
		"28cdc101137c88aedd0a709208004500002db03a4000400675590a0000be0a00007ac7ce04d22a67581700000d535018fa4bffff000068656c6c6f",
	}
	for i, data := range packets {
		data, _ := hex.DecodeString(data)
		ehdr := eth.DecodeEthernetHeader(data)
		ps := stacks.NewPortStack(stacks.PortStackConfig{
			MaxOpenPortsTCP: 1,
			MTU:             2048,
			MAC:             ehdr.Destination,
		})
		sock, err := stacks.NewTCPSocket(ps, stacks.TCPSocketConfig{})
		if err != nil {
			t.Fatal(i, err)
		}
		err = ps.OpenTCP(dataport, sock)
		if err != nil {
			t.Fatal(i, err)
		}
		err = ps.RecvEth(data)
		if err != nil && !errors.Is(err, stacks.ErrDroppedPacket) {
			t.Fatal(i, err)
		}
	}
}

type Exchanger struct {
	Stacks   []*stacks.PortStack
	pipesN   []int
	pipes    [][2048]byte
	segments []seqs.Segment
	ex       int
	loglevel slog.Level
}

func NewExchanger(stacks ...*stacks.PortStack) *Exchanger {
	egr := &Exchanger{
		Stacks: stacks,
		pipesN: make([]int, len(stacks)),
		pipes:  make([][2048]byte, len(stacks)),
		ex:     -1,
	}
	return egr
}

func (egr *Exchanger) isdebug() bool { return egr.loglevel <= slog.LevelDebug }
func (egr *Exchanger) isinfo() bool  { return egr.loglevel <= slog.LevelInfo }

// LastSegment returns the last TCP segment sent over the stack.
func (egr *Exchanger) LastSegment() seqs.Segment {
	if len(egr.segments) == 0 {
		return seqs.Segment{}
	}
	return egr.segments[len(egr.segments)-1]
}

func (egr *Exchanger) getPayload(istack int) []byte {
	return egr.pipes[istack][:egr.pipesN[istack]]
}

func (egr *Exchanger) zeroPayload(istack int) {
	egr.pipesN[istack] = 0
	egr.pipes[istack] = [2048]byte{}
}

func (egr *Exchanger) HandleTx(t *testing.T) (pkts, bytesSent int) {
	egr.ex++
	t.Helper()
	var err error
	for istack := 0; istack < len(egr.Stacks); istack++ {
		// This first for loop generates packets "in-flight" contained in `pipes` data structure.
		egr.pipesN[istack], err = egr.Stacks[istack].HandleEth(egr.pipes[istack][:])
		if (err != nil && !isDroppedPacket(err)) || egr.pipesN[istack] < 0 {
			t.Errorf("ex[%d] send[%d]: %s", egr.ex, istack, err)
			return pkts, bytesSent
		} else if isDroppedPacket(err) && egr.isdebug() {
			t.Logf("ex[%d] send[%d]: %s", egr.ex, istack, err)
		}
		if egr.pipesN[istack] > 0 {
			pkts++
			pkt, err := stacks.ParseTCPPacket(egr.getPayload(istack))
			if err == nil {
				seg := pkt.TCP.Segment(len(pkt.Payload()))
				egr.segments = append(egr.segments, seg)
				if egr.isdebug() {
					t.Logf("ex[%d] send[%d]: %+v", egr.ex, istack, seg)
				}
			}
		}
		bytesSent += egr.pipesN[istack]
	}
	return pkts, bytesSent
}

func (egr *Exchanger) HandleRx(t *testing.T) {
	var err error
	for isend := 0; isend < len(egr.Stacks); isend++ {
		// We deliver each in-flight packet to all stacks, except the one that sent it.
		payload := egr.getPayload(isend)
		if len(payload) == 0 {
			continue
		}
		for irecv := 0; irecv < len(egr.Stacks); irecv++ {
			if irecv == isend {
				continue // Don't deliver to self.
			}
			err = egr.Stacks[irecv].RecvEth(payload)
			if err != nil && !isDroppedPacket(err) {
				t.Errorf("ex[%d] recv[%d]: %s", egr.ex, irecv, err)
			} else if isDroppedPacket(err) && egr.isdebug() {
				t.Logf("ex[%d] recv[%d]: %s", egr.ex, irecv, err)
			}
		}
		egr.zeroPayload(isend)
	}
}

// DoExchanges exchanges packets between stacks until no more data is being sent or maxExchanges is reached.
// By convention client (initiator) is the first stack and server (listener) is the second when dealing with pairs.
func (egr *Exchanger) DoExchanges(t *testing.T, maxExchanges int) (exDone, bytesSent int) {
	t.Helper()
	for ; exDone < maxExchanges; exDone++ {
		pkts, bytes := egr.HandleTx(t)
		bytesSent += bytes
		if pkts == 0 {
			break // No more data being sent.
		}
		egr.HandleRx(t)
	}
	return exDone, bytesSent
}

func isDroppedPacket(err error) bool {
	return err != nil && (errors.Is(err, stacks.ErrDroppedPacket) || strings.HasPrefix(err.Error(), "drop"))
}

func createTCPClientServerPair(t *testing.T) (client, server *stacks.TCPSocket) {
	t.Helper()
	const (
		clientPort = 1025
		clientISS  = 100
		clientWND  = 1000

		serverPort = 80
		serverISS  = 300
		serverWND  = 1300
	)
	Stacks := createPortStacks(t, 2)
	clientStack := Stacks[0]
	serverStack := Stacks[1]

	// Configure server
	serverIP := netip.AddrPortFrom(serverStack.Addr(), serverPort)

	serverTCP, err := stacks.NewTCPSocket(serverStack, stacks.TCPSocketConfig{
		TxBufSize: 2048,
		RxBufSize: 2048,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = serverTCP.OpenListenTCP(serverIP.Port(), serverISS)
	// serverTCP, err := stacks.ListenTCP(serverStack, serverIP.Port(), serverISS, serverWND)
	if err != nil {
		t.Fatal(err)
	}

	// Configure client.
	clientTCP, err := stacks.NewTCPSocket(clientStack, stacks.TCPSocketConfig{
		TxBufSize: 2048,
		RxBufSize: 2048,
	})
	if err != nil {
		t.Fatal(err)
	}
	err = clientTCP.OpenDialTCP(clientPort, serverStack.MACAs6(), serverIP, clientISS)
	// clientTCP, err := stacks.DialTCP(clientStack, clientPort, Stacks[1].MACAs6(), serverIP, clientISS, clientWND)
	if err != nil {
		t.Fatal(err)
	}
	err = clientStack.FlagPendingTCP(clientPort)
	if err != nil {
		t.Fatal(err)
	}

	return clientTCP, serverTCP
}

func createPortStacks(t *testing.T, n int) (Stacks []*stacks.PortStack) {
	t.Helper()
	if n > math.MaxUint16 {
		t.Fatal("too many stacks")
	}
	for i := 0; i < n; i++ {
		u8 := [2]uint8{uint8(i) + 1, uint8(i>>8) + 1}
		MAC := [6]byte{0: u8[0], 1: u8[1]}
		ip := netip.AddrFrom4([4]byte{192, 168, u8[1], u8[0]})
		Stack := stacks.NewPortStack(stacks.PortStackConfig{
			MAC:             MAC,
			MaxOpenPortsTCP: 1,
			MaxOpenPortsUDP: 1,
			MTU:             2048,
		})
		Stack.SetAddr(ip)
		Stacks = append(Stacks, Stack)
	}
	return Stacks
}

func socketReadAllString(s *stacks.TCPSocket) string {
	var str strings.Builder
	var buf [1024]byte
	for s.BufferedInput() > 0 {
		n, err := s.ReadDeadline(buf[:], time.Time{})
		str.Write(buf[:n])
		if n == 0 || err != nil {
			break
		}
	}
	return str.String()
}

func socketSendString(s *stacks.TCPSocket, str string) {
	_, err := s.Write([]byte(str))
	if err != nil {
		panic(err)
	}
}

type multihandler func(dst []byte, rxPkt *stacks.TCPPacket) (int, error)

func (mh multihandler) handleEth(dst []byte) (n int, err error) {
	return mh(dst, nil)
}

func (mh multihandler) recvTCP(rxPkt *stacks.TCPPacket) error {
	_, err := mh(nil, rxPkt)
	return err
}

func (mh multihandler) isPendingHandling() bool {
	n, _ := mh(nil, nil)
	return n > 0
}

func setLog(ps *stacks.PortStack, group string, lvl slog.Level) {
	output := os.Stdout
	log := slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
		Level: lvl,
	}))
	ps.SetLogger(log.WithGroup(group))
}
