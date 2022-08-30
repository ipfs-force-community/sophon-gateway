FROM filvenus/venus-buildenv AS buildenv

COPY ./go.mod ./venus-gateway/go.mod
COPY ./extern/ ./venus-gateway/extern/
RUN export GOPROXY=https://goproxy.cn,direct && cd venus-gateway   && go mod download 
COPY . ./venus-gateway
RUN export GOPROXY=https://goproxy.cn,direct && cd venus-gateway  && make

RUN cd venus-gateway && ldd ./venus-gateway


FROM filvenus/venus-runtime

# copy the app from build env
COPY --from=buildenv  /go/venus-gateway/venus-gateway /app/venus-gateway

EXPOSE 45132

ENTRYPOINT ["/app/venus-gateway"]
