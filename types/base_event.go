package types

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	logging "github.com/ipfs/go-log/v2"
	"github.com/modern-go/reflect2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("gateway_stream")

type ChannelMgr interface {
	GetChannel() ([]*ChannelInfo, error)
}
type BaseEventStream struct {
	reqLk     sync.RWMutex
	idRequest map[sharedTypes.UUID]*types.RequestEvent
	cfg       *Config
}

func NewBaseEventStream(ctx context.Context, cfg *Config) *BaseEventStream {
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
		return xerrors.Errorf("send request must have channel")
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
	if len(channels) == 1 {
		return err
	}

	log.Warnf("the first channel is fail, try to other channesl")
	otherChannels := channels[1:]
	respCh := make(chan *types.ResponseEvent)
	for _, channel := range otherChannels {
		go func(channel *ChannelInfo) {
			respEvent, err := e.sendOnce(ctx, channel, method, payload)
			if err != nil {
				log.Errorf("send request %s to %s failed %v", method, channel.Ip, err)
				return
			}
			respCh <- respEvent
		}(channel)
	}

	select {
	case resp := <-respCh:
		return processResp(resp)
	case <-ctx.Done():
		return xerrors.Errorf("request cancel by context")
	}
}

func (e *BaseEventStream) sendOnce(ctx context.Context, channel *ChannelInfo, method string, payload []byte) (*types.ResponseEvent, error) {
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
	case channel.OutBound <- request:
		log.Debug("send request %s to %s", method, channel.Ip)
	case <-ctx.Done():
		return nil, xerrors.Errorf("send request cancel by context %w", ctx.Err())
	}

	//wait for result
	//timeout here
	select {
	case <-ctx.Done():
		return nil, xerrors.Errorf("cancel by context %w", ctx.Err())
	case respEvent := <-resultCh:
		return respEvent, nil
	}
}

func (e *BaseEventStream) cleanRequests(ctx context.Context) {
	tm := time.NewTicker(time.Minute * 5)
	go func() {
		for {
			select {
			case <-tm.C:
				e.reqLk.Lock()
				for id, request := range e.idRequest {
					if time.Since(request.CreateTime) > e.cfg.RequestTimeout {
						delete(e.idRequest, id)
					}
				}
				e.reqLk.Unlock()
			case <-ctx.Done():
				log.Warnf("return clean request")
				return
			}
		}

	}()
}

func (e *BaseEventStream) ResponseEvent(ctx context.Context, resp *types.ResponseEvent) error {
	e.reqLk.Lock()
	event, ok := e.idRequest[resp.ID]
	if ok {
		delete(e.idRequest, resp.ID)
	} else {
		log.Errorf("request id %s not exit %v", resp.ID.String(), resp)
	}
	e.reqLk.Unlock()
	if ok {
		event.Result <- resp
	}
	return nil
}
