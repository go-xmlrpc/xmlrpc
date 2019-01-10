package xmlrpc

import (
	"encoding/xml"

	"github.com/pkg/errors"
)

type MethodCall struct {
	XMLName xml.Name `xml:"methodCall"`

	Name   string        `xml:"methodName"`
	Params []interface{} `xml:"params>param,omitempty"`
}

type IntParam struct {
	Value int `xml:"value>int"`
}

type StringParam struct {
	Value string `xml:"value>string"`
}

type DoubleParam struct {
	Value float64 `xml:"value>double"`
}

type ArrayParam struct {
	Values []interface{} `xml:"value>array>data"`
}

func NewParam(data interface{}) (interface{}, error) {
	// TODO: Allow this to be extended if the type passed in implements the
	// proper Marshal and Unmarshal methods.

	switch v := data.(type) {
	case int:
		return IntParam{Value: v}, nil
	case float32:
		return DoubleParam{Value: float64(v)}, nil
	case float64:
		return DoubleParam{Value: v}, nil
	case string:
		return StringParam{Value: v}, nil
	default:
		return nil, errors.Errorf("unknown param type: %T", v)
	}
}

func Marshal(name string, args ...interface{}) ([]byte, error) {
	out := &MethodCall{
		Name: name,
	}
	for _, arg := range args {
		param, err := NewParam(arg)
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
