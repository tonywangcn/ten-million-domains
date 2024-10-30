FROM --platform=linux/amd64 golang:1.23.2 AS builder
ENV USER=appuser
ENV UID=10001 
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "${UID}" \    
    "${USER}"
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o main .

FROM --platform=linux/amd64 gcr.io/distroless/static-debian11
LABEL org.opencontainers.image.source https://github.com/tonywangcn/ten-million-domains
ENV ENV=PROD
WORKDIR /app
COPY --from=builder /app/main ./
COPY --from=builder /app/data /app/data