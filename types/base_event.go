package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	logging "github.com/ipfs/go-log/v2"
	"github.com/modern-go/reflect2"
)

var log = logging.Logger("gateway_stream")

var ErrCloseChannel = fmt.Errorf("channel closed")
var ErrRequestTimeout = fmt.Errorf("timer clean this request due to exceed wait time")

type BaseEventStream struct {
	reqLk     sync.RWMutex
	idRequest map[sharedTypes.UUID]*types.RequestEvent
	cfg       *RequestConfig
}

func NewBaseEventStream(ctx context.Context, cfg *RequestConfig) *BaseEventStream {
	baseEventStream := &BaseEventStream{
		reqLk:     sync.RWMutex{},
		idRequest: make(map[sharedTypes.UUID]*types.RequestEvent),
		cfg:       cfg,
	}
	go baseEventStream.cleanRequests(ctx)
	return baseEventStream
}

func (e *BaseEventStream) SendRequest(ctx context.Context, channels []*ChannelInfo, method string, payload []byte, result interface{}) error {
	if len(channels) == 0 {
		return fmt.Errorf("send request must have channel")
	}

	processResp := func(resp *types.ResponseEvent) error {
		if len(resp.Error) > 0 {
			return errors.New(resp.Error)
		}

		if !reflect2.IsNil(result) {
			return json.Unmarshal(resp.Payload, result)
		}
		return nil
	}
	firstChanel := channels[0]
	resp, err := e.sendOnce(ctx, firstChanel, method, payload)
	if err == nil {
		return processResp(resp)
	}

	if ctx.Err() != nil || len(channels) == 1 || isTimeoutError(err) { // if ctx have done before, not to try others
		return err
	}

	// code below unable to work as expect , because there no way to detect network issue in gateway,
	log.Warnf("the first channel is fail, try to other channel")

	var lk sync.Mutex
	var respOnce sync.Once
	otherChannels := channels[1:]
	respCh := make(chan *types.ResponseEvent)
	errRespCount := 0
	for _, channel := range otherChannels {
		go func(channel *ChannelInfo) {
			respEvent, err := e.sendOnce(ctx, channel, method, payload)
			if err != nil {
				lk.Lock()
				errRespCount++
				log.Errorf("send request %s failed %v", method, err)
				// all requests failed
				if errRespCount == len(otherChannels) {
					respCh <- &types.ResponseEvent{
						Error: fmt.Sprintf("all request failed: %s %v", method, err),
					}
				}
				lk.Unlock()
				return
			}
			// only send once, avoid goroutine leak
			respOnce.Do(func() {
				respCh <- respEvent
			})
		}(channel)
	}

	select {
	case resp := <-respCh:
		return processResp(resp)
	case <-ctx.Done():
		return fmt.Errorf("request cancel by context")
	}
}

func (e *BaseEventStream) sendOnce(ctx context.Context, channel *ChannelInfo, method string, payload []byte) (response *types.ResponseEvent, err error) {
	id := sharedTypes.NewUUID()
	resultCh := make(chan *types.ResponseEvent, 1)
	request := &types.RequestEvent{
		ID:         id,
		Method:     method,
		Payload:    payload,
		CreateTime: time.Now(),
		Result:     resultCh,
	}
	e.reqLk.Lock()
	e.idRequest[id] = request
	e.reqLk.Unlock()

	select {
	case <-channel.Ctx.Done():
		return nil, ErrCloseChannel
	case <-ctx.Done():
		return nil, fmt.Errorf("send request cancel by context %w", ctx.Err())
	default:
		channel.OutBound <- request //NOTICE this may be panic, but will catch by recover and try other, should never have  other panic
		log.Debug("send request %s to %s", method, channel.Ip)
	}

	// wait for result
	// timeout here
	select {
	case <-channel.Ctx.Done():
		return nil, ErrCloseChannel
	case <-ctx.Done():
		return nil, fmt.Errorf("cancel by context %w", ctx.Err())
	case respEvent := <-resultCh:
		return respEvent, nil
	}
}

func (e *BaseEventStream) cleanRequests(ctx context.Context) {
	tm := time.NewTicker(e.cfg.ClearInterval)
	for {
		select {
		case <-tm.C:
			e.reqLk.Lock()
			for id, request := range e.idRequest {
				if time.Since(request.CreateTime) > e.cfg.RequestTimeout {
					delete(e.idRequest, id)
					// avoid block this channel, maybe client request come as request timeout by chance
					select {
					case request.Result <- &types.ResponseEvent{
						ID:      id,
						Payload: nil,
						Error:   fmt.Errorf("%w %s method %s", ErrRequestTimeout, request.CreateTime, request.Method).Error(),
					}:
					default:
					}
				}
			}
			e.reqLk.Unlock()
		case <-ctx.Done():
			log.Warnf("return clean request")
			return
		}
	}
}

func (e *BaseEventStream) ResponseEvent(ctx context.Context, resp *types.ResponseEvent) error {
	e.reqLk.Lock()
	event, ok := e.idRequest[resp.ID]
	if ok {
		delete(e.idRequest, resp.ID)
	} else {
		e.reqLk.Unlock()
		return fmt.Errorf("request id %s not exit", resp.ID.String())
	}
	e.reqLk.Unlock()
	if ok {
		select {
		case event.Result <- resp:
		default:
		}
	}
	return nil
}

func isTimeoutError(err error) bool {
	if !reflect2.IsNil(err) {
		return strings.Contains(err.Error(), ErrRequestTimeout.Error())
	}
	return false
}
