package framing

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
)

// FrameFNF is fire and forget frame.
type FrameFNF struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameFNF) Validate() (err error) {
	return
}

func (p *FrameFNF) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("FrameFNF{%s,data=%s,metadata=%s}", p.header, p.DataUTF8(), m)
}

// Metadata returns metadata bytes.
func (p *FrameFNF) Metadata() ([]byte, bool) {
	return p.trySliceMetadata(0)
}

// Data returns data bytes.
func (p *FrameFNF) Data() []byte {
	return p.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p *FrameFNF) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *FrameFNF) DataUTF8() string {
	return string(p.Data())
}

// NewFrameFNF returns a new fire and forget frame.
func NewFrameFNF(sid uint32, data, metadata []byte, flags ...FrameFlag) *FrameFNF {
	fg := newFlags(flags...)
	bf := common.BorrowByteBuffer()
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	_, _ = bf.Write(data)
	return &FrameFNF{
		&BaseFrame{
			header: NewFrameHeader(sid, FrameTypeRequestFNF, fg),
			body:   bf,
		},
	}
}
