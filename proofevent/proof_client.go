package proofevent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/sophon-gateway/types"
)

type ProofEvent struct {
	client       v2API.IProofServiceProvider
	mAddr        address.Address
	proofHandler types.ProofHandler
	log          *zap.SugaredLogger
	readyCh      chan struct{}
}

func NewProofRegisterClient(ctx context.Context, url, token string) (v2API.IProofServiceProvider, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+token)
	client, closer, err := v2API.NewIGatewayRPC(ctx, url, headers)
	if err != nil {
		return nil, nil, err
	}
	return client, closer, nil
}

func NewProofEvent(client v2API.IProofServiceProvider, mAddr address.Address, proofHandler types.ProofHandler, log *zap.SugaredLogger) *ProofEvent {
	return &ProofEvent{
		client:       client,
		mAddr:        mAddr,
		proofHandler: proofHandler,
		log:          log,
		readyCh:      make(chan struct{}, 1),
	}
}

func (e *ProofEvent) WaitReady(ctx context.Context) {
	select {
	case <-e.readyCh:
	case <-ctx.Done():
	}
}

func (e *ProofEvent) ListenProofRequest(ctx context.Context) {
	e.log.Infof("start proof event listening")
	for {
		if err := e.listenProofRequestOnce(ctx); err != nil {
			e.log.Errorf("listen proof request errored: %s", err)
		} else {
			e.log.Warn("listenProofRequest quit")
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			e.log.Warnf("not restarting listenProofRequest: context error: %s", ctx.Err())
			return
		}

		e.log.Info("restarting listenProofRequest")
		// try clear ready channel
		select {
		case <-e.readyCh:
		default:
		}
	}
}

func (e *ProofEvent) listenProofRequestOnce(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	policy := &gateway.ProofRegisterPolicy{
		MinerAddress: e.mAddr,
	}
	proofEventCh, err := e.client.ListenProofEvent(ctx, policy)
	if err != nil {
		// Retry is handled by caller
		return fmt.Errorf("listenProofRequest failed: %w", err)
	}

	for proofEvent := range proofEventCh {
		switch proofEvent.Method {
		case "InitConnect":
			req := gateway.ConnectedCompleted{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				return fmt.Errorf("odd error in connect %v", err)
			}
			e.readyCh <- struct{}{}
			e.log.Infof("success to connect with proof %s", req.ChannelId)
		case "ComputeProof":
			req := gateway.ComputeProofRequest{}
			err := json.Unmarshal(proofEvent.Payload, &req)
			if err != nil {
				e.error(ctx, proofEvent.ID, err)
				continue
			}
			e.processComputeProof(ctx, proofEvent.ID, req)
		default:
			e.log.Errorf("unexpect proof event type %s", proofEvent.Method)
		}
	}

	return nil
}

// context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version
func (e *ProofEvent) processComputeProof(ctx context.Context, reqId sharedTypes.UUID, req gateway.ComputeProofRequest) {
	proof, err := e.proofHandler.ComputeProof(ctx, req.SectorInfos, req.Rand, req.Height, req.NWVersion)
	if err != nil {
		e.error(ctx, reqId, err)
		return
	}
	e.value(ctx, reqId, proof)
}

func (e *ProofEvent) value(ctx context.Context, id sharedTypes.UUID, val interface{}) {
	respBytes, err := json.Marshal(val)
	if err != nil {
		e.log.Errorf("marshal address list error %s", err)
		e.error(ctx, id, err)
		return
	}
	err = e.client.ResponseProofEvent(ctx, &gateway.ResponseEvent{
		ID:      id,
		Payload: respBytes,
		Error:   "",
	})
	if err != nil {
		e.log.Errorf("response error %v", err)
	}
}

func (e *ProofEvent) error(ctx context.Context, id sharedTypes.UUID, err error) {
	err = e.client.ResponseProofEvent(ctx, &gateway.ResponseEvent{
		ID:      id,
		Payload: nil,
		Error:   err.Error(),
	})
	if err != nil {
		e.log.Errorf("response error %v", err)
	}
}
