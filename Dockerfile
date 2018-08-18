FROM golang:1.10 as builder

COPY main.go ./
RUN go get github.com/coreos/go-iptables/iptables
RUN go get k8s.io/client-go/...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /main main.go

FROM alpine:3.7

RUN apk add --no-cache iptables
COPY --from=builder /main .
RUN ["chmod", "+x", "/main"]
CMD ["./main"]
