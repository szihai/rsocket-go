package rsocket

import (
	"io"
)

type FrameFNF struct {
	*Header
	metadata []byte
	data     []byte
}

func (p *FrameFNF) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	if p.Header.Flags().Check(FlagMetadata) {
		wrote, err = w.Write(encodeU24(len(p.metadata)))
		n += int64(wrote)
		if err != nil {
			return
		}
	}
	if p.data != nil {
		wrote, err = w.Write(p.data)
		n += int64(wrote)
	}
	return
}

func (p *FrameFNF) Size() int {
	size := headerLen
	if p.Header.Flags().Check(FlagMetadata) {
		size += 3 + len(p.metadata)
	}
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FrameFNF) Metadata() []byte {
	return p.metadata
}

func (p *FrameFNF) Data() []byte {
	return p.data
}

func asFNF(h *Header, raw []byte) *FrameFNF {
	m, d := sliceMetadataAndData(h, raw, headerLen)
	return &FrameFNF{
		Header:   h,
		metadata: m,
		data:     d,
	}
}
