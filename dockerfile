FROM filvenus/venus-buildenv AS buildenv

COPY . ./venus-gateway
RUN export GOPROXY=https://goproxy.cn && cd venus-gateway  && make

RUN cd venus-gateway && ldd ./venus-gateway


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the app from build env
COPY --from=buildenv  /go/venus-gateway/venus-gateway /app/venus-gateway



EXPOSE 45132

ENTRYPOINT ["/app/venus-gateway"]
