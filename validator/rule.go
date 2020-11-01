package validator

import (
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"html"
	"reflect"
	"regexp"
	"strings"

	"github.com/zzztttkkk/suna/utils"
)

const (
	_Bool = iota
	_Int64
	_Uint64
	_Float64
	_Bytes
	_String

	_JsonObject
	_JsonArray

	_BoolSlice
	_IntSlice
	_UintSlice
	_FloatSlice
	_StringSlice
	_BytesSlice
)

var typeNames = []string{
	"Bool",
	"Int",
	"Uint",
	"Float",
	"String",
	"String",

	"JsonObject",
	"JsonArray",

	"BoolArray",
	"IntArray",
	"UintArray",
	"FloatArray",
	"StringArray",
	"StringArray",
}

type _Rule struct {
	form     string // request form name
	field    string // struct field name
	path     string // request uservalue name
	t        int    // form type
	required bool
	info     string // print to message when form error

	vrange bool // int value value range

	minVF bool  // min int value validate flag
	minV  int64 // min int value
	maxVF bool  // max int value validate flag
	maxV  int64 // max int value

	minUVF bool
	minUV  uint64
	maxUVF bool
	maxUV  uint64

	minFVF bool
	minFV  float64
	maxFVF bool
	maxFV  float64

	lrange bool // bytes value length range
	minLF  bool
	minL   int64
	maxLF  bool
	maxL   int64

	srange bool // slice value size range
	minSF  bool
	minS   int64
	maxSF  bool
	maxS   int64

	defaultV []byte

	reg     *regexp.Regexp
	regName string

	fn     func([]byte) ([]byte, bool)
	fnName string

	isSlice bool
}

var ruleFmt = utils.NewNamedFmt(
	"|${name}|${path}|${type}|${required}|${lrange}|${vrange}|${srange}|${default}|${regexp}|${function}|${descp}|",
)

//revive:disable:cyclomatic
func (rule *_Rule) String() string {
	m := utils.M{
		"name":     rule.form,
		"type":     typeNames[rule.t],
		"required": rule.required,
		"descp":    rule.info,
		"path":     rule.path,
	}
	if len(rule.info) < 1 {
		m["descp"] = "/"
	}
	if rule.vrange {
		switch rule.t {
		case _Int64, _IntSlice:
			if rule.minVF && rule.maxVF {
				m["vrange"] = fmt.Sprintf("%d-%d", rule.minV, rule.maxV)
			} else if rule.minVF {
				m["vrange"] = fmt.Sprintf("%d-", rule.minV)
			} else if rule.maxVF {
				m["vrange"] = fmt.Sprintf("-%d", rule.maxV)
			} else {
				m["vrange"] = "/"
			}
		case _Uint64, _UintSlice:
			if rule.minUVF && rule.maxUVF {
				m["vrange"] = fmt.Sprintf("%d-%d", rule.minUV, rule.maxUV)
			} else if rule.minUVF {
				m["vrange"] = fmt.Sprintf("%d-", rule.minUV)
			} else if rule.maxUVF {
				m["vrange"] = fmt.Sprintf("-%d", rule.maxUV)
			} else {
				m["vrange"] = "/"
			}
		case _Float64, _FloatSlice:
			if rule.minUVF && rule.maxUVF {
				m["vrange"] = fmt.Sprintf("%d-%d", rule.minUV, rule.maxUV)
			} else if rule.minUVF {
				m["vrange"] = fmt.Sprintf("%d-", rule.minUV)
			} else if rule.maxUVF {
				m["vrange"] = fmt.Sprintf("-%d", rule.maxUV)
			} else {
				m["vrange"] = "/"
			}
		}
	} else {
		m["vrange"] = "/"
	}

	if rule.lrange {
		if rule.minLF && rule.maxLF {
			m["lrange"] = fmt.Sprintf("%d-%d", rule.minL, rule.maxL)
		} else if rule.minLF {
			m["lrange"] = fmt.Sprintf("%d-", rule.minL)
		} else if rule.maxUVF {
			m["lrange"] = fmt.Sprintf("-%d", rule.maxL)
		} else {
			m["lrange"] = "/"
		}
	} else {
		m["lrange"] = "/"
	}

	if rule.srange {
		if rule.minSF && rule.maxSF {
			m["srange"] = fmt.Sprintf("%d-%d", rule.minS, rule.maxS)
		} else if rule.minSF {
			m["srange"] = fmt.Sprintf("%d-", rule.minS)
		} else if rule.maxSF {
			m["srange"] = fmt.Sprintf("-%d", rule.maxS)
		} else {
			m["srange"] = "/"
		}
	} else {
		m["srange"] = "/"
	}

	if rule.reg != nil {
		m["regexp"] = fmt.Sprintf(
			`<code class="regexp" descp="%s">%s</code>`,
			html.EscapeString(fmt.Sprintf("`%s`", rule.reg.String())),
			html.EscapeString(rule.regName),
		)
	} else {
		m["regexp"] = "/"
	}

	if len(rule.defaultV) > 0 {
		m["default"] = html.EscapeString(string(rule.defaultV))
	} else {
		m["default"] = "/"
	}

	if rule.fnName != "" {
		m["function"] = fmt.Sprintf(
			`<code class="function" descp="%s">%s</code>`,
			html.EscapeString(funcDescp[rule.fnName]),
			html.EscapeString(rule.fnName),
		)
	} else {
		m["function"] = "/"
	}
	return ruleFmt.Render(m)
}

func (rule *_Rule) peek(ctx *fasthttp.RequestCtx) []byte {
	var val []byte
	if len(rule.path) > 0 {
		_v, ok := ctx.UserValue(rule.path).([]byte)
		if ok {
			val = _v
		}
	} else {
		val = ctx.FormValue(rule.form)
	}

	if len(val) > 0 {
		val = bytes.TrimSpace(val)
	}
	if len(val) == 0 && len(rule.defaultV) > 0 {
		val = rule.defaultV
	}
	return val
}

type _RuleSliceT []*_Rule

func (a _RuleSliceT) Len() int      { return len(a) }
func (a _RuleSliceT) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a _RuleSliceT) Less(i, j int) bool {
	l, r := a[i], a[j]
	if l.required != r.required {
		if l.required {
			return true
		}
	}
	return a[i].form < a[j].form
}

// markdown table
func (a _RuleSliceT) String() string {
	buf := strings.Builder{}
	buf.WriteString("|name|path param|type|required|length range|value range|size range|default|regexp|function|description|\n")
	buf.WriteString("|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|:---:|\n")
	for _, r := range a {
		buf.WriteString(r.String())
		buf.WriteByte('\n')
	}
	return buf.String()
}

type Rules struct {
	lst    _RuleSliceT
	isJson bool
	raw    reflect.Type
}

func (rs *Rules) NewDoc(descp string) *Doc {
	ele := reflect.New(rs.raw).Elem()
	fnV := ele.MethodByName("Descprition")
	if fnV.IsValid() {
		txt := (fnV.Call(nil))[0].Interface().(string)
		if len(descp) > 0 {
			descp = fmt.Sprintf("%s\n%s", descp, txt)
		} else {
			descp = txt
		}
	}
	return &Doc{
		descp:  descp,
		fields: rs.lst.String(),
	}
}

type Doc struct {
	descp  string
	fields string
}

func (d *Doc) Document() string {
	return fmt.Sprintf("### description\n%s\n### fields\n%s\n", d.descp, d.fields)
}
