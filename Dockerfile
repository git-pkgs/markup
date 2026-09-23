FROM golang:1.25-bookworm

# Python docutils for reStructuredText (provides rst2html)
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3-docutils \
    perl \
    && rm -rf /var/lib/apt/lists/*

# Ruby and asciidoctor for AsciiDoc
RUN gem install asciidoctor --no-document

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build ./...
RUN go test ./...
