FROM filvenus/venus-buildenv AS buildenv

COPY . ./venus-gateway
RUN export GOPROXY=https://goproxy.cn && cd venus-gateway  && make

RUN cd venus-gateway && ldd ./venus-gateway


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the app from build env
COPY --from=buildenv  /go/venus-gateway/venus-gateway /app/venus-gateway


# copy ddl
COPY --from=buildenv  /usr/lib/x86_64-linux-gnu/libhwloc.so.15 \
    /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 \
    /lib/x86_64-linux-gnu/libgcc_s.so.1 \
    /lib/x86_64-linux-gnu/libutil.so.1 \
    /lib/x86_64-linux-gnu/librt.so.1 \
    /lib/x86_64-linux-gnu/libpthread.so.0 \
    /lib/x86_64-linux-gnu/libm.so.6 \
    /lib/x86_64-linux-gnu/libdl.so.2 \
    /lib/x86_64-linux-gnu/libc.so.6 \
    /usr/lib/x86_64-linux-gnu/libnuma.so.1 \
    /usr/lib/x86_64-linux-gnu/libltdl.so.7 \
    /lib/

COPY ./docker/script  /script

EXPOSE 45132

ENTRYPOINT ["/app/venus-gateway"]
