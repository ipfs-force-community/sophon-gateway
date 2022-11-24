// stm: #unit
package types

import (
	"context"
	"encoding/json"
	"fmt"
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
		// stm: @VENUSGATEWAY_TYPES_RESPONSE_EVENT_001
		go client.start(ctx)
		clients = append(clients, client)
		getConns := func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}

		// stm: @VENUSGATEWAY_TYPES_SEND_REQUEST_001
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.NoError(t, err)

		// stm: @VENUSGATEWAY_TYPES_SEND_REQUEST_002
		err = eventSteam.SendRequest(ctx, nil, "mock_method", parms, result)
		require.Error(t, err)
	})

	// test for bug https://github.com/filecoin-project/venus/issues/4992
	t.Run("fix deadlock", func(t *testing.T) {
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
		getConns := func() []*ChannelInfo {
			var channels []*ChannelInfo
			for _, client := range clients {
				channels = append(channels, client.channel)
			}
			return channels
		}
		err = eventSteam.ResponseEvent(ctx, &types.ResponseEvent{
			ID:      sharedTypes.NewUUID(),
			Payload: nil,
			Error:   "",
		})
		require.Error(t, err)
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
		getConns := func() []*ChannelInfo {
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
		getConns := func() []*ChannelInfo {
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
		getConns := func() []*ChannelInfo {
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

		getConns := func() []*ChannelInfo {
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

		getConns := func() []*ChannelInfo {
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
		var requests []*types.RequestEvent
		eventSteam.reqLk.Lock()
		for i := 0; i < 10; i++ {
			req := &types.RequestEvent{
				CreateTime: time.Now(),
				Result:     make(chan *types.ResponseEvent, 1),
			}
			eventSteam.idRequest[sharedTypes.NewUUID()] = req
			requests = append(requests, req)
		}
		eventSteam.reqLk.Unlock()
		go eventSteam.cleanRequests(ctx)
		time.Sleep(time.Second * 5)
		eventSteam.reqLk.Lock()
		require.Len(t, eventSteam.idRequest, 0)
		require.Len(t, eventSteam.idRequest, 0)
		for _, req := range requests {
			require.Len(t, req.Result, 1)
			result := <-req.Result
			require.Contains(t, result.Error, ErrRequestTimeout.Error())
		}
		eventSteam.reqLk.Unlock()
	})

	t.Run("all request failed", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eventSteam := NewBaseEventStream(ctx, DefaultConfig())

		// expect id not exists error
		// stm: @VENUSGATEWAY_TYPES_RESPONSE_EVENT_002
		err := eventSteam.ResponseEvent(ctx, &types.ResponseEvent{ID: sharedTypes.NewUUID()})
		require.Error(t, err)

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
		client2.close()
		err = eventSteam.SendRequest(ctx, getConns(), "mock_method", parms, result)
		require.Error(t, err)
		require.Contains(t, err.Error(), "all request failed:")
	})
}

func TestIstimeOutError(t *testing.T) {
	err := fmt.Errorf("%w %s method %s", ErrRequestTimeout, time.Now(), "MOCK")
	require.True(t, isTimeoutError(err))
}

type mockClient struct {
	t              *testing.T
	event          *BaseEventStream
	requestCh      chan *types.RequestEvent
	channel        *ChannelInfo
	delayToReponse time.Duration

	closeCh   chan struct{}
	waitClose chan struct{}

	cancel context.CancelFunc
}

func setupClient(t *testing.T, event *BaseEventStream, ip string) *mockClient {
	requestCh := make(chan *types.RequestEvent)
	ctx, cancel := context.WithCancel(context.Background())

	return &mockClient{
		t:         t,
		requestCh: requestCh,
		event:     event,
		channel:   NewChannelInfo(ctx, ip, requestCh),
		closeCh:   make(chan struct{}),
		waitClose: make(chan struct{}),
		cancel:    cancel,
	}
}

func (m *mockClient) close() {
	m.cancel()
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
