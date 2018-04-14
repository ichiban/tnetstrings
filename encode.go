package tnetstrings

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Encoder is a streaming tnetstrings encoder.
type Encoder struct {
	io.Writer
}

// Encode encodes a value into tnetstring.
func (e *Encoder) Encode(val interface{}) error {
	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.String:
		s := val.(string)
		_, err := fmt.Fprintf(e, "%d:%s,", len(s), s)
		return err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		s := fmt.Sprintf("%d", val)
		_, err := fmt.Fprintf(e, "%d:%s#", len(s), s)
		return err
	case reflect.Float32, reflect.Float64:
		s := fmt.Sprintf("%f", val)
		_, err := fmt.Fprintf(e, "%d:%s^", len(s), s)
		return err
	case reflect.Bool:
		s := strconv.FormatBool(val.(bool))
		_, err := fmt.Fprintf(e, "%d:%s!", len(s), s)
		return err
	case reflect.Invalid:
		_, err := fmt.Fprint(e, "0:~")
		return err
	case reflect.Ptr:
		v = v.Elem()
		return e.Encode(v.Interface())
	case reflect.Map:
		return e.encodeMap(v)
	case reflect.Struct:
		return e.encodeStruct(v)
	case reflect.Array, reflect.Slice:
		return e.encodeSlice(v)
	}
	return ErrUnsupportedType{Type: v.Type()}
}

func (e *Encoder) encodeMap(v reflect.Value) error {
	var buf bytes.Buffer
	f := Encoder{
		Writer: &buf,
	}
	ks := v.MapKeys()
	sort.Slice(ks, func(i, j int) bool {
		return ks[i].String() < ks[j].String()
	})
	for _, k := range ks {
		if err := f.Encode(k.Interface()); err != nil {
			return err
		}
		if err := f.Encode(v.MapIndex(k).Interface()); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(e, "%d:%s}", buf.Len(), buf.Bytes())
	return err
}

func (e *Encoder) encodeStruct(v reflect.Value) error {
	var buf bytes.Buffer
	f := Encoder{
		Writer: &buf,
	}
	for i := 0; i < v.NumField(); i++ {
		name, val, ok := field(v, i)
		if !ok {
			continue
		}

		if err := f.Encode(name); err != nil {
			return err
		}
		if err := f.Encode(val); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(e, "%d:%s}", buf.Len(), buf.Bytes())
	return err
}

func field(v reflect.Value, i int) (string, interface{}, bool) {
	t := v.Type().Field(i)
	fv := v.Field(i)

	if !fv.CanInterface() {
		return "", nil, false
	}

	name := t.Name
	if tag, ok := t.Tag.Lookup("tnetstrings"); ok {
		if tag == "-" {
			return "", nil, false
		}
		ts := strings.Split(tag, ",")
		if len(ts) > 0 && ts[0] != "" {
			name = ts[0]
		}
		if len(ts) > 1 && ts[1] == "omitempty" && fv == reflect.Zero(t.Type) {
			return "", nil, false
		}
	}
	return name, fv.Interface(), true
}

func (e *Encoder) encodeSlice(v reflect.Value) error {
	if v.Type().Elem().Kind() == reflect.Uint8 {
		s := fmt.Sprintf("%s", v.Interface())
		_, err := fmt.Fprintf(e, "%d:%s,", len(s), s)
		return err
	}
	var buf bytes.Buffer
	f := Encoder{
		Writer: &buf,
	}
	for i := 0; i < v.Len(); i++ {
		if err := f.Encode(v.Index(i).Interface()); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(e, "%d:%s]", buf.Len(), buf.Bytes())
	return err
}
