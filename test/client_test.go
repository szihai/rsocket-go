package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

var client rsocket.ClientSocket

func init() {
	socket, err := rsocket.Connect().
		SetupPayload(payload.NewString("hello", "world")).
		MetadataMimeType("application/json").
		DataMimeType("application/json").
		KeepAlive(3*time.Second, 2*time.Second, 3).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) rx.Mono {
					return rx.JustMono(payload.NewString("foo", "bar"))
				}),
				rsocket.RequestStream(func(msg payload.Payload) rx.Flux {
					log.Println("receive:", msg)
					return rx.Range(0, 10).
						Map(func(n int) payload.Payload {
							return payload.NewString(fmt.Sprintf("from_golang_%d", n), "stream")
						})
				}),
				rsocket.RequestChannel(func(msgs rx.Publisher) rx.Flux {
					rx.ToFlux(msgs).
						SubscribeOn(rx.ElasticScheduler()). // <-- use elastic scheduler, DO NOT block here!
						DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
							log.Println("receive channel:", elem)
						})
					return rx.Range(0, 10).
						Map(func(n int) payload.Payload {
							return payload.NewString(fmt.Sprintf("from_golang_%d", n), "channel")
						})
				}),
			)
		}).
		Transport("tcp://127.0.0.1:7878").
		Start()
	if err != nil {
		log.Fatal(err)
	}
	client = socket
	logger.Infof("+++++ CONNECT SUCCESS +++++\n")

	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	go func() {
		<-done
		_ = client.Close()
		log.Println("socket closed")
	}()
}

func TestClient_MetadataPush(t *testing.T) {
	client.MetadataPush(payload.NewString("hello", "world"))
}

func TestClient_FireAndForget(t *testing.T) {
	client.FireAndForget(payload.NewString("hello", "world"))
}

func TestClient_RequestResponse(t *testing.T) {
	client.RequestResponse(payload.NewString("hello", "world")).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("oops...", err)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("rcv:", elem)
			assert.Equal(t, "hello", elem.DataUTF8())
			metadata, _ := elem.MetadataUTF8()
			assert.Equal(t, "world", metadata)
		}).
		Subscribe(context.Background())
}

func TestClient_RequestStream(t *testing.T) {
	done := make(chan struct{})

	var totals int

	c := 7

	client.RequestStream(payload.NewString("hello", fmt.Sprintf("%d", c))).
		LimitRate(3).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			close(done)
		}).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("oops...", err)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			time.Sleep(500 * time.Millisecond)
			log.Println("rcv:", elem)
			assert.Equal(t, fmt.Sprintf("hello_%d", totals), elem.DataUTF8(), "bad data")
			metadata, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d", c), metadata, "bad metadata")
			totals++
		}).
		Subscribe(context.Background())
	<-done
}

func TestClient_RequestChannel(t *testing.T) {
	//logger.SetLoggerLevel(logger.LogLevelDebug)
	done := make(chan struct{})
	sending := rx.Range(0, 10).Map(func(n int) payload.Payload {
		return payload.NewString("h", "b")
	})
	client.
		RequestChannel(sending).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("next:", elem)
		}).
		Subscribe(context.Background())
	<-done
}
