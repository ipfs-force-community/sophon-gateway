package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc/auth"
	auth2 "github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/core"
	"github.com/ipfs-force-community/venus-gateway/types"
	"gopkg.in/resty.v1"
	"net"
	"net/http"
	"strings"
	"time"
)

type VenusAuthHandler struct {
	Verify func(spanId, serviceName, preHost, host, token string) (*auth2.VerifyResponse, error)
	Next   http.HandlerFunc
}

func MacAddr() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		panic("net interfaces" + err.Error())
	}
	mac := ""
	for _, netInterface := range interfaces {
		mac = netInterface.HardwareAddr.String()
		if len(mac) == 0 {
			continue
		}
		break
	}
	return mac
}

func (h *VenusAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := r.Header.Get("Authorization")
	ctx = context.WithValue(ctx, types.IPKey, r.RemoteAddr)
	// if other nodes on the same PC, the permission check will passes directly
	if strings.Split(r.RemoteAddr, ":")[0] == "127.0.0.1" {
		ctx = auth.WithPerm(ctx, []auth.Permission{"read", "write", "sign", "admin"})
	} else {
		if token == "" {
			token = r.FormValue("token")
			if token != "" {
				token = "Bearer " + token
			}
		}
		if token != "" {
			if !strings.HasPrefix(token, "Bearer ") {
				log.Warn("missing Bearer prefix in venusauth header")
				w.WriteHeader(401)
				return
			}
			token = strings.TrimPrefix(token, "Bearer ")
			res, err := h.Verify(MacAddr(), "lotus", r.RemoteAddr, r.Host, token)
			if err != nil {
				log.Warnf("JWT Verification failed (originating from %s): %s", r.RemoteAddr, err)
				w.WriteHeader(401)
				return
			}
			ctx = context.WithValue(ctx, types.AccountKey, res.Name)
			perms := core.AdaptOldStrategy(res.Perm)
			perms2 := make([]auth.Permission, 0)
			for _, v := range perms {
				perms2 = append(perms2, auth.Permission(v))
			}
			ctx = auth.WithPerm(ctx, perms2)
		}
	}
	h.Next(w, r.WithContext(ctx))
}

type JWTClient struct {
	cli *resty.Client
}

func NewJWTClient(url string) *JWTClient {
	client := resty.New().
		SetHostURL(url).
		SetRetryCount(0).
		SetTimeout(time.Second).
		SetHeader("Accept", "application/json")

	return &JWTClient{
		cli: client,
	}
}

// Verify: post method for Verify token
// @spanId: local service unique Id
// @serviceName: e.g. venus
// @preHost: the IP of the request server
// @host: local service IP
// @token: jwt token gen from this service
func (c *JWTClient) Verify(spanId, serviceName, preHost, host, token string) (*auth2.VerifyResponse, error) {
	response, err := c.cli.R().SetHeader("X-Forwarded-For", host).
		SetHeader("X-Real-Ip", host).
		SetHeader("spanId", spanId).
		SetHeader("preHost", preHost).
		SetHeader("svcName", serviceName).
		SetHeader("Origin", host).
		SetFormData(map[string]string{
			"token": token,
		}).Post("/verify")
	if err != nil {
		return nil, err
	}
	switch response.StatusCode() {
	case http.StatusOK:
		var res = new(auth2.VerifyResponse)
		response.Body()
		err = json.Unmarshal(response.Body(), res)
		return res, err
	default:
		response.Result()
		return nil, fmt.Errorf("response code is : %d, msg:%s", response.StatusCode(), response.Body())
	}
}
