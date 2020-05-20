package colorjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const initialDepth = 0
const valueSep = ","
const null = "null"
const startMap = "{"
const endMap = "}"
const startArray = "["
const endArray = "]"

const emptyMap = startMap + endMap
const emptyArray = startArray + endArray

type Formatter struct {
	KeyColor        *color.Color
	StringColor     *color.Color
	BoolColor       *color.Color
	NumberColor     *color.Color
	NullColor       *color.Color
	StringMaxLength int
	Indent          int
	DisabledColor   bool
	RawStrings      bool
}

func NewFormatter() *Formatter {
	return &Formatter{
		KeyColor:        color.New(color.FgWhite),
		StringColor:     color.New(color.FgGreen),
		BoolColor:       color.New(color.FgYellow),
		NumberColor:     color.New(color.FgCyan),
		NullColor:       color.New(color.FgMagenta),
		StringMaxLength: 0,
		DisabledColor:   false,
		Indent:          0,
		RawStrings:      false,
	}
}

func (f *Formatter) sprintfColor(c *color.Color, format string, args ...interface{}) string {
	if f.DisabledColor || c == nil {
		return fmt.Sprintf(format, args...)
	}
	return c.SprintfFunc()(format, args...)
}

func (f *Formatter) sprintColor(c *color.Color, s string) string {
	if f.DisabledColor || c == nil {
		return fmt.Sprint(s)
	}
	return c.SprintFunc()(s)
}

func (f *Formatter) writeIndent(buf *bytes.Buffer, depth int) {
	buf.WriteString(strings.Repeat(" ", f.Indent*depth))
}

func (f *Formatter) writeObjSep(buf *bytes.Buffer) {
	if f.Indent != 0 {
		buf.WriteByte('\n')
	} else {
		buf.WriteByte(' ')
	}
}

func (f *Formatter) Marshal(jsonObj interface{}) []byte {
	buffer := bytes.Buffer{}
	f.marshalValue(jsonObj, &buffer, initialDepth)
	return buffer.Bytes()
}

func (f *Formatter) marshalMap(m map[string]interface{}, buf *bytes.Buffer, depth int) {
	remaining := len(m)

	if remaining == 0 {
		buf.WriteString(emptyMap)
		return
	}

	keys := make([]string, 0)
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	buf.WriteString(startMap)
	f.writeObjSep(buf)

	for _, key := range keys {
		f.writeIndent(buf, depth+1)
		buf.WriteString(f.KeyColor.Sprintf("\"%s\": ", key))
		f.marshalValue(m[key], buf, depth+1)
		remaining--
		if remaining != 0 {
			buf.WriteString(valueSep)
		}
		f.writeObjSep(buf)
	}
	f.writeIndent(buf, depth)
	buf.WriteString(endMap)
}

func (f *Formatter) marshalArray(a []interface{}, buf *bytes.Buffer, depth int) {
	if len(a) == 0 {
		buf.WriteString(emptyArray)
		return
	}

	buf.WriteString(startArray)
	f.writeObjSep(buf)

	for i, v := range a {
		f.writeIndent(buf, depth+1)
		f.marshalValue(v, buf, depth+1)
		if i < len(a)-1 {
			buf.WriteString(valueSep)
		}
		f.writeObjSep(buf)
	}
	f.writeIndent(buf, depth)
	buf.WriteString(endArray)
}

func (f *Formatter) marshalValue(val interface{}, buf *bytes.Buffer, depth int) {
	// TODO: use reflect !!!!
	switch v := val.(type) {
	case map[string]interface{}:
		f.marshalMap(v, buf, depth)
	case http.Header:
		var tmp = make(map[string]interface{})
		for k, val := range v {
			tmp[k] = append(val[:0:0], val...)
		}
		f.marshalValue(tmp, buf, depth)
	case []interface{}:
		f.marshalArray(v, buf, depth)
	case []string:
		var tmp = []interface{}{}
		for _, v := range v {
			tmp = append(tmp, v)
		}
		f.marshalValue(tmp, buf, depth)
	case string:
		f.marshalString(v, buf)
	case float64:
		buf.WriteString(f.sprintColor(f.NumberColor, strconv.FormatFloat(v, 'f', -1, 64)))
	case bool:
		buf.WriteString(f.sprintColor(f.BoolColor, (strconv.FormatBool(v))))
	case nil:
		buf.WriteString(f.sprintColor(f.NullColor, null))
	}
}

func (f *Formatter) marshalString(str string, buf *bytes.Buffer) {
	if !f.RawStrings {
		strBytes, _ := json.Marshal(str)
		str = string(strBytes)
	}

	if f.StringMaxLength != 0 && len(str) >= f.StringMaxLength {
		str = fmt.Sprintf("%s...", str[0:f.StringMaxLength])
	}

	buf.WriteString(f.sprintColor(f.StringColor, str))
}

// Marshal JSON data with default options
func Marshal(jsonObj interface{}) []byte {
	return NewFormatter().Marshal(jsonObj)
}
