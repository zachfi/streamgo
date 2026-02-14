FROM alpine:3.23
COPY ./bin/linux/streamgo /bin/streamgo
RUN chmod 0700 /bin/streamgo
RUN mkdir /var/streamgo
RUN apk --update add ca-certificates
RUN apk add tzdata
ENTRYPOINT ["/bin/streamgo"]
