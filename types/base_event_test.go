package types

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/stretchr/testify/require"
)

type mockParams struct {
	A string
}

type mockResult struct {
	B string
}

func TestSendRequest(t *testing.T) {
	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		client := setupClient(t, eventSteam, "127.1.1.1")
		go client.start(ctx)
		clients = append(clients, client)
		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.NoError(t, err)
	})

	t.Run("send multiple", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		for i := 0; i < 10; i++ {
			client := setupClient(t, eventSteam, "127.1.1.1")
			go client.start(ctx)
			clients = append(clients, client)
		}
		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.NoError(t, err)
	})

	t.Run("send multi", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.EqualError(t, err, "send request must have channel")
	})

	t.Run("once send error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		client := setupClient(t, eventSteam, "127.1.1.1")
		go client.start(ctx)
		clients = append(clients, client)
		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		sendCtx, sendCancel := context.WithCancel(context.Background())
		sendCancel()
		err = eventSteam.SendRequest(sendCtx, getConns(), "mock_method", parms, result)
		require.EqualError(t, err, "send request cancel by context context canceled")
	})

	t.Run("once send error and retry others", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		client := setupClient(t, eventSteam, "127.1.1.1")
		go client.start(ctx)
		clients = append(clients, client)

		client2 := setupClient(t, eventSteam, "127.1.1.2")
		go client2.start(ctx)
		clients = append(clients, client2)

		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		client.close()
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.NoError(t, err)
	})

	t.Run("multiple cancel error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		parms, err := json.Marshal(mockParams{A: "mock arg"})
		require.NoError(t, err)
		result := &mockResult{}

		var clients []*mockClient
		for i := 0; i < 10; i++ {
			client := setupClient(t, eventSteam, "127.1.1.1")
			go client.start(ctx)
			clients = append(clients, client)
		}

		var getConns = func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		sendCtx, sendCancel := context.WithCancel(context.Background())
		sendCancel()
		err = eventSteam.SendRequest(sendCtx, getConns(), "mock_method", parms, result)
		require.EqualError(t, err, "send request cancel by context context canceled")
	})

	t.Run("clear timeout request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, &RequestConfig{
			RequestQueueSize: 30,
			RequestTimeout:   time.Second,
			ClearInterval:    time.Second,
		})
		for i := 0; i < 10; i++ {
			eventSteam.idRequest[sharedTypes.NewUUID()] = &types.RequestEvent{
				CreateTime: time.Now(),
			}
		}
		eventSteam.cleanRequests(ctx)
		<-time.After(time.Second * 5)
		require.Len(t, eventSteam.idRequest, 0)
	})
}

type mockClient struct {
	t              *testing.T
	event          *BaseEventStream
	requestCh      chan *types.RequestEvent
	channel        *ChannelInfo
	delayToReponse time.Duration

	closeCh   chan struct{}
	waitClose chan struct{}
}

func setupClient(t *testing.T, event *BaseEventStream, ip string) *mockClient {
	requestCh := make(chan *types.RequestEvent)

	return &mockClient{
		t:         t,
		requestCh: requestCh,
		event:     event,
		channel:   NewChannelInfo(ip, requestCh),
		closeCh:   make(chan struct{}),
		waitClose: make(chan struct{}),
	}
}

func (m *mockClient) close() {
	close(m.requestCh)
	m.closeCh <- struct{}{}
	<-m.waitClose
}

func (m *mockClient) start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
		case <-m.closeCh:
			m.waitClose <- struct{}{}
		case req, ok := <-m.requestCh:
			if ok {
				time.Sleep(m.delayToReponse)
				var params mockParams
				err := json.Unmarshal(req.Payload, &params)
				require.NoError(m.t, err)
				require.Equal(m.t, "mock arg", params.A)
				result := mockResult{
					B: "mock",
				}
				data, err := json.Marshal(result)
				require.NoError(m.t, err)
				err = m.event.ResponseEvent(ctx, &types.ResponseEvent{
					ID:      req.ID,
					Payload: data,
					Error:   "",
				})
				require.NoError(m.t, err)
			}
		}
	}
}
