package xmlrpc

import (
	"encoding/xml"

	"github.com/pkg/errors"
)

type methodCall struct {
	XMLName xml.Name `xml:"methodCall"`

	Name   string        `xml:"methodName"`
	Params []interface{} `xml:"params>param,omitempty"`
}

type intParam struct {
	Value int `xml:"value>int"`
}

type stringParam struct {
	Value string `xml:"value>string"`
}

type doubleParam struct {
	Value float64 `xml:"value>double"`
}

type arrayParam struct {
	Values []interface{} `xml:"value>array>data"`
}

func newParam(data interface{}) (interface{}, error) {
	// TODO: Add support for additional types

	switch v := data.(type) {
	case int:
		return intParam{Value: v}, nil
	case float32:
		return doubleParam{Value: float64(v)}, nil
	case float64:
		return doubleParam{Value: v}, nil
	case string:
		return stringParam{Value: v}, nil
	default:
		return nil, errors.Errorf("unknown param type: %T", v)
	}
}

func Marshal(name string, args ...interface{}) ([]byte, error) {
	out := &methodCall{
		Name: name,
	}
	for _, arg := range args {
		param, err := newParam(arg)
		if err != nil {
			return nil, errors.Wrap(err, "xmlrpc: marshal")
		}
		out.Params = append(out.Params, param)
	}

	data, err := xml.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "xmlrpc: marshal")
	}

	return data, nil
}
