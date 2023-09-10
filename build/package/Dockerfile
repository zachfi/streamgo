FROM alpine:3.18
COPY streamgo /bin/streamgo
RUN mkdir /var/streamgo
RUN apk --update add ca-certificates
RUN apk add tzdata
ENTRYPOINT ["/bin/streamgo"]
