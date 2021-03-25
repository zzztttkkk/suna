package sha

import (
	"bytes"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/zzztttkkk/sha/utils"
	"io"
	"sync"
)

var (
	headerCompressValueSep = []byte(", ")
	gzipStr                = []byte("gzip")
	deflateStr             = []byte("deflate")
	brotliStr              = []byte("br")

	CompressLevelGzip    = gzip.DefaultCompression
	CompressLevelDeflate = flate.DefaultCompression
	CompressLevelBrotli  = brotli.DefaultCompression
)

type _CompressionWriter interface {
	io.Writer
	Flush() error
	Reset(writer io.Writer)
}

var brWriterPool = sync.Pool{New: func() interface{} { return nil }}

func (ctx *RequestCtx) CompressBrotli() {
	ctx.Response.Header().Set(HeaderContentEncoding, brotliStr)
	var cwr *brotli.Writer
	brI := brWriterPool.Get()
	w := &ctx.Response._HTTPPocket

	if brI != nil {
		cwr = brI.(*brotli.Writer)
		cwr.Reset(w)
	} else {
		cwr = brotli.NewWriterLevel(w, CompressLevelBrotli)
	}
	ctx.Response.cw = cwr
	ctx.Response.cwp = &brWriterPool
}

var gzipWriterPool = sync.Pool{New: func() interface{} { return nil }}

func (ctx *RequestCtx) CompressGzip() {
	ctx.Response.Header().Set(HeaderContentEncoding, gzipStr)

	var cwr *gzip.Writer
	var err error
	w := &ctx.Response._HTTPPocket

	brI := gzipWriterPool.Get()
	if brI != nil {
		cwr = brI.(*gzip.Writer)
		cwr.Reset(w)
	} else {
		cwr, err = gzip.NewWriterLevel(w, CompressLevelGzip)
		if err != nil {
			panic(err)
		}
	}
	ctx.Response.cw = cwr
	ctx.Response.cwp = &gzipWriterPool
}

var deflateWriterPool = sync.Pool{New: func() interface{} { return nil }}

func (ctx *RequestCtx) CompressDeflate() {
	ctx.Response.Header().Set(HeaderContentEncoding, deflateStr)

	var cwr *flate.Writer
	var err error
	w := &ctx.Response._HTTPPocket
	brI := deflateWriterPool.Get()
	if brI != nil {
		cwr = brI.(*flate.Writer)
		cwr.Reset(w)
	} else {
		cwr, err = flate.NewWriter(w, CompressLevelDeflate)
		if err != nil {
			panic(err)
		}
	}
	ctx.Response.cw = cwr
	ctx.Response.cwp = &deflateWriterPool
}

var disableCompress = false

// DisableCompress keep raw response body when debugging
func DisableCompress() {
	disableCompress = true
}

// br > deflate > gzip
func (ctx *RequestCtx) AutoCompress() {
	if disableCompress {
		return
	}

	acceptGzip := false
	acceptDeflate := false

	for _, headerVal := range ctx.Request.Header().GetAll(HeaderAcceptEncoding) {
		for _, v := range bytes.Split(headerVal, headerCompressValueSep) {
			switch utils.S(v) {
			case "gzip":
				acceptGzip = true
			case "deflate":
				acceptDeflate = true
			case "br":
				ctx.CompressBrotli()
				return
			}
		}
	}

	if acceptDeflate {
		ctx.CompressDeflate()
		return
	}

	if acceptGzip {
		ctx.CompressGzip()
	}
}
