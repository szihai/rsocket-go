package rsocket

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/common"
)

const epsilon = 1e-4

type socketSupplier struct {
	mutex *sync.Mutex

	u string
	b ClientBuilder

	tau             int64
	stamp           int64
	errorPercentage common.Ewma
}

func (p *socketSupplier) String() string {
	return fmt.Sprintf("SocketSupplier{transport=%s, v=%.2f}", p.u, p.availability())
}

func (p *socketSupplier) create() (socket *availabilitySocket, err error) {
	var v float64
	var origin ClientSocket
	origin, err = p.b.clone().Transport(p.u).Start()
	if err == nil {
		socket = newWeightedSocket(origin, p)
		v = 1
	}
	p.updateErrorPercentage(v)
	return
}

func (p *socketSupplier) availability() float64 {
	e := p.errorPercentage.Value()
	if common.NowInMicrosecond()-p.stamp > p.tau {
		a := math.Min(1.0, e+0.5)
		p.mutex.Lock()
		p.errorPercentage.Reset(a)
		p.mutex.Unlock()
	}
	if e < epsilon {
		e = 0
	} else if 1-epsilon < e {
		e = 1
	}
	return e
}

func (p *socketSupplier) updateErrorPercentage(v float64) {
	p.mutex.Lock()
	p.errorPercentage.Insert(v)
	p.stamp = common.NowInMicrosecond()
	p.mutex.Unlock()
}

func newSocketSupplier(builder ClientBuilder, uri string) *socketSupplier {
	return &socketSupplier{
		mutex:           &sync.Mutex{},
		u:               uri,
		b:               builder,
		tau:             common.CalcTAU(5, time.Second),
		stamp:           common.NowInMicrosecond(),
		errorPercentage: common.NewEwma(5, time.Second, 1.0),
	}
}
