FROM golang:1.25.5-alpine3.23 AS builder

RUN apk add --update nodejs npm && node --version && npm --version && npm install -g pnpm@latest-10
RUN GOBIN=/usr/local/bin go install github.com/bufbuild/buf/cmd/buf@v1.61.0

WORKDIR /build
COPY . .

# gen protobuf files
RUN cd ui && buf lint && rm -rf src/gen/* && buf generate

# build ui
RUN export CI=true &&  cd ui && pnpm i && pnpm build

# build backend
RUN go build -o flowstate ./app/

FROM public.ecr.aws/docker/library/alpine:3.23

COPY --from=builder /build/flowstate /flowstate

CMD ["/flowstate"]
