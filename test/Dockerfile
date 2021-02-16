FROM golang:latest
WORKDIR /app
RUN go get github.com/stretchr/testify
RUN go get github.com/Trendyol/gaos/runner
COPY . .
CMD ["go", "test"]