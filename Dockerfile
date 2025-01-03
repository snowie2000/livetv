FROM alpine:latest
WORKDIR /opt
COPY ./livetv /opt/livetv 
RUN apk --no-cache add ca-certificates tzdata libc6-compat libstdc++

EXPOSE 9000
VOLUME ["/opt/data"]
CMD ["/opt/livetv"]