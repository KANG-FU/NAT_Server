package testing

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"

	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cs438/peer"
	"go.dedis.ch/cs438/registry"
	"go.dedis.ch/cs438/registry/standard"

	"go.dedis.ch/cs438/transport"
	"go.dedis.ch/cs438/types"
)

// NewFakeMessage return a new fake message.
func NewFakeMessage(t *testing.T) FakeMessage {
	content := make([]byte, 12)

	_, err := rand.Read(content)
	require.NoError(t, err)

	return FakeMessage{
		Content: content,
	}
}

// FakeMessage defines a fake message that can be used over the network.
//
// - implements types.Message.
type FakeMessage struct {
	Content []byte
}

// NewEmpty implements types.Message.
func (m FakeMessage) NewEmpty() types.Message {
	return &FakeMessage{}
}

// Name implements types.Message.
func (m FakeMessage) Name() string {
	return "fake"
}

// String implements types.Message.
func (m FakeMessage) String() string {
	return fmt.Sprintf("{fake:%x}", m.Content)
}

// HTML implements types.Message.
func (m FakeMessage) HTML() string {
	return m.String()
}

// GetNetMsg return the net.Message representation of the message.
func (m FakeMessage) GetNetMsg(t *testing.T) transport.Message {
	buf, err := json.Marshal(&m)
	require.NoError(t, err)

	return transport.Message{
		Type:    m.Name(),
		Payload: buf,
	}
}

// Compare compare the fake message to a net.Message.
func (m FakeMessage) Compare(t *testing.T, msg *transport.Message) {
	require.Equal(t, m.Name(), msg.Type)

	var newMsg FakeMessage

	err := json.Unmarshal(msg.Payload, &newMsg)
	require.NoError(t, err)

	require.Equal(t, m.Content, newMsg.Content)
}

// GetHandler returns a handler that check the content of the received message
// and closes the channel.
func (m FakeMessage) GetHandler(t *testing.T) (registry.Exec, Status) {
	status := NewStatus()

	return func(msg types.Message, pkt transport.Packet) error {
		defer func() {
			status.Call()
		}()

		fake, ok := msg.(*FakeMessage)
		require.True(t, ok)

		require.Equal(t, m.Content, fake.Content)

		return nil
	}, status
}

// FakeByContent sorts fake message by content
type FakeByContent []*FakeMessage

func (r FakeByContent) Len() int {
	return len(r)
}

func (r FakeByContent) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r FakeByContent) Less(i, j int) bool {
	return bytes.Compare(r[i].Content, r[j].Content) < 0
}

type configTemplate struct {
	messages []types.Message
	handlers []registry.Exec

	registry registry.Registry

	withWatcher bool
	autoStart   bool
}

func newConfigTemplate() configTemplate {
	return configTemplate{
		withWatcher: false,
		autoStart:   true,

		messages: make([]types.Message, 0),
		handlers: make([]registry.Exec, 0),

		registry: standard.NewRegistry(),
	}
}

// Option is the type of option when creating a test node.
type Option func(*configTemplate)

// WithAutostart sets the autostart option.
func WithAutostart(autostart bool) Option {
	return func(ct *configTemplate) {
		ct.autoStart = autostart
	}
}

// WithMessage will register the provided message and handler on the node.
func WithMessage(m types.Message, handler registry.Exec) Option {
	return func(ct *configTemplate) {
		ct.messages = append(ct.messages, m)
		ct.handlers = append(ct.handlers, handler)
	}
}

// WithMessageRegistry sets a specific message registry. Used to pass a proxy registry.
func WithMessageRegistry(r registry.Registry) Option {
	return func(ct *configTemplate) {
		ct.registry = r
	}
}

// NewTestNode returns a new test node.
func NewTestNode(t *testing.T, f peer.Factory, trans transport.Transport,
	addr string, opts ...Option) TestNode {

	template := newConfigTemplate()
	for _, opt := range opts {
		opt(&template)
	}

	socket, err := trans.CreateSocket(addr)
	require.NoError(t, err)

	config := peer.Configuration{}

	config.Socket = socket
	config.MessageRegistry = template.registry

	node := f(config)

	require.Equal(t, len(template.messages), len(template.handlers))
	for i, msg := range template.messages {
		config.MessageRegistry.RegisterMessageCallback(msg, template.handlers[i])
	}

	if template.autoStart {
		err := node.Start()
		require.NoError(t, err)
	}

	return TestNode{
		Peer:   node,
		config: config,
		socket: socket,
		t:      t,
	}
}

// TestNode defines a test node. It overides peer.Peer with additional functions
// for testing.
type TestNode struct {
	peer.Peer
	config peer.Configuration
	socket transport.ClosableSocket
	t      *testing.T
}

// GetAddr returns the node's socket address
func (t *TestNode) GetAddr() string {
	return t.socket.GetAddress()
}

// StopAll stops the peer and socket.
func (t *TestNode) StopAll() {
	t.Peer.Stop()
	err := t.socket.Close()
	require.NoError(t.t, err)
}

// GetIns returns all the messages received so far.
func (t TestNode) GetIns() []transport.Packet {
	return t.socket.GetIns()
}

// GetOuts returns all the messages sent so far.
func (t TestNode) GetOuts() []transport.Packet {
	return t.socket.GetOuts()
}

// GetRegistry returns the node's registry
func (t TestNode) GetRegistry() registry.Registry {
	return t.config.MessageRegistry
}

// GetFakes filters out all the processed messages of type FakeMessage
func (t TestNode) GetFakes() []*FakeMessage {
	msgs := t.config.MessageRegistry.GetMessages()

	fakes := make([]*FakeMessage, 0)

	for _, msg := range msgs {
		fake, ok := msg.(*FakeMessage)
		if ok {
			fakes = append(fakes, fake)
		}
	}

	return fakes
}

// GetChatMsgs filters out all the processed messages of type ChatMessage
func (t TestNode) GetChatMsgs() []*types.ChatMessage {
	msgs := t.config.MessageRegistry.GetMessages()

	chatMsgs := make([]*types.ChatMessage, 0)

	for _, msg := range msgs {
		chatMsg, ok := msg.(*types.ChatMessage)
		if ok {
			chatMsgs = append(chatMsgs, chatMsg)
		}
	}

	return chatMsgs
}

// Status allows to check if something has been called or not.
type Status struct {
	called chan struct{}
}

// NewStatus return a new initialized Status.
func NewStatus() Status {
	return Status{
		called: make(chan struct{}),
	}
}

// Call notifies that the status has been called.
func (s Status) Call() {
	select {
	case <-s.called:
	default:
		close(s.called)
	}
}

// CheckCalled checks if the status has been called.
func (s Status) CheckCalled(t *testing.T) {
	select {
	case <-s.called:
	default:
		t.Error("has not been called")
	}
}

// CheckNotCalled checks if the status has been called.
func (s Status) CheckNotCalled(t *testing.T) {
	select {
	case <-s.called:
		t.Error("has been called")
	default:
	}
}

// GetChat returns the ChatMessage associated to the transport.Message.
func GetChat(t *testing.T, msg *transport.Message) types.ChatMessage {
	require.Equal(t, "chat", msg.Type)

	var chatMessage types.ChatMessage

	err := json.Unmarshal(msg.Payload, &chatMessage)
	require.NoError(t, err)

	return chatMessage
}

// GetRandBytes returns random bytes.
func GetRandBytes(t *testing.T) []byte {
	res := make([]byte, 12)

	_, err := rand.Read(res)
	require.NoError(t, err)

	return res
}

// GetRandString returns a random string.
func GetRandString() string {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

	res := make([]byte, 12)
	for i := range res {
		res[i] = charset[rand.Intn(len(charset))]
	}

	return string(res)
}

// GetRandomPkt return a packet containing randomly filled fields.
func GetRandomPkt(t *testing.T) transport.Packet {
	pkt := transport.Packet{
		Header: &transport.Header{
			PacketID:    GetRandString(),
			TTL:         uint(rand.Int()),
			Timestamp:   rand.Int63(),
			Source:      GetRandString(),
			RelayedBy:   GetRandString(),
			Destination: GetRandString(),
		},
		Msg: &transport.Message{
			Type:    GetRandString(),
			Payload: []byte(fmt.Sprintf(`{"data":"%s"}`, GetRandString())),
		},
	}

	return pkt
}

// GetSocket is a handy function to create a socket and check for error
func GetSocket(t *testing.T, transp transport.Transport, addr string) transport.ClosableSocket {
	sock, err := transp.CreateSocket(addr)
	require.NoError(t, err)

	return sock
}

// Terminable describes a peer that have a terminate function. Which is the case
// if this is a binnode.
type Terminable interface {
	Terminate() error
}
