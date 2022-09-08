FROM node:lts-alpine as frontend

WORKDIR /frontend-build

COPY ./web/package*.json ./
RUN npm install

COPY ./web .
RUN npm run build

FROM golang:1.16-alpine as backend

RUN apk add --no-cache gcc musl-dev linux-headers

WORKDIR /backend-build

COPY go.* ./
RUN go mod download

COPY . .
COPY --from=frontend /frontend-build/dist ./web/dist

RUN go build -o shibc-faucet -ldflags "-s -w"

FROM alpine

RUN apk add --no-cache ca-certificates

COPY --from=backend /backend-build/shibc-faucet /app/shibc-faucet

EXPOSE 8080

ENTRYPOINT ["/app/shibc-faucet"]
