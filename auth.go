package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc/auth"
	auth2 "github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/core"
	"github.com/ipfs-force-community/venus-gateway/types"
	"go.opencensus.io/trace"
	"net/http"
	"strings"
)

type VenusAuthHandler struct {
	Verify func(ctx context.Context, token string) (*auth2.VerifyResponse, error)
	Next   http.HandlerFunc
}

func jwtUserFromToken(token string) (string, error) {
	sks := strings.Split(token, ".")
	if len(sks) != 3 {
		return "", fmt.Errorf("invalid token")

	}

	enc := []byte(sks[1])
	encoding := base64.RawURLEncoding
	dec := make([]byte, encoding.DecodedLen(len(enc)))
	if _, err := encoding.Decode(dec, enc); err != nil {
		return "", err
	}
	payload := &auth2.JWTPayload{}
	err := json.Unmarshal(dec, payload)
	return payload.Name, err
}

func (h *VenusAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, span := trace.StartSpan(r.Context(), "VenusAuthHandler.ServeHTTP",
		func(so *trace.StartOptions) { so.Sampler = trace.AlwaysSample() })
	defer span.End()

	token := r.Header.Get("Authorization")
	if token == "" {
		token = r.FormValue("token")
		if token != "" {
			token = "Bearer " + token
		}
	}

	if len(token) == 0 {
		// local call doesn't need a token
		if strings.Split(r.RemoteAddr, ":")[0] == "127.0.0.1" {
			ctx = auth.WithPerm(ctx, []auth.Permission{"read", "write", "sign", "admin"})
		} else {
			message := "JWT verifycation failed, empty token"
			span.SetStatus(trace.Status{Code: trace.StatusCodeUnauthenticated, Message: message})
			log.Warnf(message)
			w.WriteHeader(401)
			return
		}
	}

	ctx = context.WithValue(ctx, types.IPKey, h.getClientIp(r))

	// if other nodes on the same PC, the permission check will passes directly
	if token != "" {
		if !strings.HasPrefix(token, "Bearer ") {
			log.Warn("missing Bearer prefix in venus-auth header")
			w.WriteHeader(401)
			return
		}
		token = strings.TrimPrefix(token, "Bearer ")

		if mayUser, _ := jwtUserFromToken(token); len(mayUser) != 0 {
			span.AddAttributes(trace.StringAttribute("Account-Unverified", mayUser))
		}

		span.AddAttributes(trace.StringAttribute("X-Real-IP", r.RemoteAddr),
			trace.StringAttribute("preHost", r.Host))

		res, err := h.Verify(ctx, token)

		if err != nil {
			message := fmt.Sprintf("JWT Verification failed (originating from %s): %s", r.RemoteAddr, err.Error())
			span.SetStatus(trace.Status{
				Code:    trace.StatusCodeUnauthenticated,
				Message: message})
			log.Warnf(message)
			w.WriteHeader(401)
			return
		}

		span.AddAttributes(trace.StringAttribute("Account", res.Name))

		ctx = context.WithValue(ctx, types.AccountKey, res.Name)
		perms := core.AdaptOldStrategy(res.Perm)
		ctx = auth.WithPerm(ctx, append([]auth.Permission{}, perms...))
	}

	h.Next(w, r.WithContext(ctx))
}

func (h *VenusAuthHandler) getClientIp(r *http.Request) string {
	realIp := r.Header.Get("X-Real-IP")
	if len(realIp) == 0 {
		return r.RemoteAddr
	} else {
		return realIp
	}
}

/*type JWTClient struct {
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
*/
