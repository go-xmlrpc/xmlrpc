package xmlrpc

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type Fault struct {
	FaultCode   int
	FaultString string
}

func (f Fault) Error() string {
	return fmt.Sprintf("fault (%d): %s", f.FaultCode, f.FaultString)
}

func Decode(r io.Reader) ([]interface{}, error) {
	// TODO: Handle fault responses

	d := xml.NewDecoder(r)

	_, err := nextProcInst(d)
	if err != nil {
		return nil, err
	}

	// Find the first start token
	_, err = nextStartToken(d, "methodResponse")
	if err != nil {
		return nil, err
	}

	// Read in the response
	params, err := decodeMethodResponse(d)
	if err != nil {
		return nil, err
	}

	for {
		// Ensure a normal token read at the end results in an io.EOF
		t, err := d.Token()
		if err == io.EOF {
			// io.EOF is a successful end
			return params, nil
		}
		if err != nil {
			return nil, err
		}

		switch t.(type) {
		case xml.CharData:
		default:
			return nil, errors.Errorf("xmlrpc Decode: unexpected token type: %T", t)
		}
	}
}

func decodeMethodResponse(d *xml.Decoder) ([]interface{}, error) {
	var params []interface{}

	t, err := nextStartToken(d, "")
	if err != nil {
		return nil, err
	}

	switch t.Name.Local {
	case "params":
		params, err = decodeParams(d)
		if err != nil {
			return nil, err
		}
	case "fault":
		err = decodeFault(d)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("xmlrpc decodeMethodResponse: unknown tag %q", t.Name.Local)
	}

	_, err = nextEndToken(d, "methodResponse")
	if err != nil {
		return nil, err
	}

	return params, nil
}

func decodeParams(d *xml.Decoder) ([]interface{}, error) {
	var params []interface{}

	for {
		t, err := nextStartOrEndToken(d, "param", "params")
		if err != nil {
			return nil, err
		}

		// If we got an EndElement, we've found all our values.
		if _, ok := t.(xml.EndElement); ok {
			return params, nil
		}

		param, err := decodeParam(d)
		if err != nil {
			return nil, err
		}

		params = append(params, param)
	}
}

func decodeParam(d *xml.Decoder) (interface{}, error) {
	_, err := nextStartToken(d, "value")
	if err != nil {
		return nil, err
	}

	ret, err := decodeValue(d)
	if err != nil {
		return nil, err
	}

	_, err = nextEndToken(d, "param")
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func decodeValue(d *xml.Decoder) (interface{}, error) {
	var ret interface{}

	sv, err := nextStartToken(d, "")
	if err != nil {
		return nil, err
	}

	switch sv.Name.Local {
	case "base64":
		ret, err = decodeBase64(d)
	case "dateTime.iso8601":
		ret, err = decodeDate(d, time.RFC3339)
	case "struct":
		ret, err = decodeStruct(d)
	case "double":
		ret, err = decodeDouble(d)
	case "int", "i4":
		ret, err = decodeInteger(d)
	case "string":
		// TODO: Handle other string format
		ret, err = decodeString(d)
	case "array":
		ret, err = decodeArray(d)
	case "nil":
		ret, err = decodeNil(d)
	case "boolean":
		ret, err = decodeBoolean(d)
	default:
		return nil, errors.Errorf("xmlrpc decodeValue: unknown inner tag %q", sv.Name.Local)
	}

	if err != nil {
		return nil, err
	}

	_, err = nextEndToken(d, "value")
	if err != nil {
		return nil, err
	}

	return ret, err
}

func decodeBase64(d *xml.Decoder) ([]byte, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return nil, err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "base64")
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(string(cd))
}

func decodeStruct(d *xml.Decoder) (map[string]interface{}, error) {
	var ret = make(map[string]interface{})

	for {
		t, err := nextStartOrEndToken(d, "member", "struct")
		if err != nil {
			return nil, err
		}

		// If we got an EndElement, we've found all our values.
		if _, ok := t.(xml.EndElement); ok {
			return ret, nil
		}

		key, val, err := decodeMember(d)
		if err != nil {
			return nil, err
		}

		if _, ok := ret[key]; ok {
			return nil, errors.Errorf("xmlrpc decodeStruct: duplicate key %q", key)
		}

		ret[key] = val
	}
}

func decodeMember(d *xml.Decoder) (string, interface{}, error) {
	var (
		key string
		val interface{}

		seenKey bool
		seenVal bool
	)

	for {
		t, err := nextStartOrEndToken(d, "", "member")
		if err != nil {
			return "", nil, err
		}

		// If we got an EndElement, we've found all our values.
		if _, ok := t.(xml.EndElement); ok {
			if !seenKey {
				return "", nil, errors.Errorf("xmlrpc decodeMember: missing name")
			}

			if !seenVal {
				return "", nil, errors.Errorf("xmlrpc decodeMember: missing value")
			}

			return key, val, nil
		}

		st := t.(xml.StartElement)
		switch st.Name.Local {
		case "name":
			if seenKey {
				return "", nil, errors.Errorf("xmlrpc decodeMember: multiple name tags")
			}

			key, err = decodeName(d)
			seenKey = true
		case "value":
			if seenVal {
				return "", nil, errors.Errorf("xmlrpc decodeMember: multiple value tags")
			}

			val, err = decodeValue(d)
			seenVal = true
		default:
			return "", nil, errors.Errorf("xmlrpc decodeMember: unknown tag %q", st.Name.Local)
		}

		if err != nil {
			return "", nil, err
		}
	}
}

func decodeName(d *xml.Decoder) (string, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return "", err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "name")
	if err != nil {
		return "", err
	}

	return string(cd), nil
}

func decodeDate(d *xml.Decoder, layout string) (time.Time, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return time.Time{}, err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "base64")
	if err != nil {
		return time.Time{}, err
	}

	return time.Parse(layout, string(cd))
}

func decodeDouble(d *xml.Decoder) (float64, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return 0, err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "double")
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(string(cd), 64)
}

func decodeInteger(d *xml.Decoder) (int, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return 0, err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	t, err := nextEndToken(d, "")
	if err != nil {
		return 0, err
	}

	if t.Name.Local != "int" && t.Name.Local != "i4" {
		return 0, errors.Errorf("xmlrpc decodeInteger: unknown closing tag %q", t.Name.Local)
	}

	return strconv.Atoi(string(cd))
}

func decodeBoolean(d *xml.Decoder) (bool, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return false, err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "boolean")
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(string(cd))
}

func decodeNil(d *xml.Decoder) (interface{}, error) {
	// Expect an end string token
	_, err := nextEndToken(d, "nil")
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func decodeString(d *xml.Decoder) (string, error) {
	cd, err := nextCharToken(d)
	if err != nil {
		return "", err
	}

	// We need to ensure we have a copy of the string data.
	cd = cd.Copy()

	// Expect an end string token
	_, err = nextEndToken(d, "string")
	if err != nil {
		return "", err
	}

	return string(cd), nil
}

func decodeArray(d *xml.Decoder) ([]interface{}, error) {
	_, err := nextStartToken(d, "data")
	if err != nil {
		return nil, err
	}

	// Decode the enclosing data tag
	ret, err := decodeData(d)
	if err != nil {
		return nil, err
	}

	// Expect an end array token
	_, err = nextEndToken(d, "array")
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func decodeData(d *xml.Decoder) ([]interface{}, error) {
	var ret []interface{}

	for {
		t, err := nextStartOrEndToken(d, "value", "data")
		if err != nil {
			return nil, err
		}

		// If we got an EndElement, we've found all our values.
		if _, ok := t.(xml.EndElement); ok {
			return ret, nil
		}

		val, err := decodeValue(d)
		if err != nil {
			return nil, err
		}

		ret = append(ret, val)
	}
}

func decodeFault(d *xml.Decoder) error {
	_, err := nextStartToken(d, "value")
	if err != nil {
		return err
	}

	val, err := decodeValue(d)
	if err != nil {
		return err
	}

	data, ok := val.(map[string]interface{})
	if !ok {
		return errors.Errorf("xmlrpc decodeFault: invalid fault data")
	}

	faultCodeRaw, ok := data["faultCode"]
	if !ok {
		return errors.Errorf("xmlrpc decodeFault: missing faultCode")
	}

	faultCode, ok := faultCodeRaw.(int)
	if !ok {
		return errors.Errorf("xmlrpc decodeFault: wrong faultCode type")
	}

	faultStringRaw, ok := data["faultString"]
	if !ok {
		return errors.Errorf("xmlrpc decodeFault: missing faultString")
	}

	faultString, ok := faultStringRaw.(string)
	if !ok {
		return errors.Errorf("xmlrpc decodeFault: wrong faultString type")
	}

	_, err = nextEndToken(d, "fault")
	if err != nil {
		return err
	}

	return Fault{
		FaultCode:   faultCode,
		FaultString: faultString,
	}
}
