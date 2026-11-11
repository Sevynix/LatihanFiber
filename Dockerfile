# syntax=docker/dockerfile:1

# =========================================================================
# Tahap 1: build. Image besar berisi Go compiler, HANYA untuk compile.
# =========================================================================
FROM golang:1.26-alpine AS build
WORKDIR /src

# go.mod dan go.sum disalin lebih dulu, terpisah dari kode. Selama dua file
# ini tidak berubah, layer "go mod download" dipakai ulang dari cache.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 menghasilkan binary statis. -trimpath membuang path folder
# komputer pembuat dari binary.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app .

# =========================================================================
# Tahap 2: runtime. Hanya binary, tanpa compiler dan tanpa source code.
# =========================================================================
FROM alpine:3.24

# Aplikasi TIDAK dijalankan sebagai root.
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/app ./app

# Logger menulis ke logs/app.log dan logs/mail.log, maka foldernya harus
# dapat ditulis user app.
RUN mkdir logs && chown app:app logs

USER app
EXPOSE 3000
ENTRYPOINT ["./app"]
CMD ["-role=api"]