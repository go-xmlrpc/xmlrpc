package xmlrpc

import (
	"encoding/xml"

	"github.com/pkg/errors"
)

func nextStartToken(d *xml.Decoder, name string) (xml.StartElement, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return xml.StartElement{}, err
		}

		switch v := t.(type) {
		case xml.StartElement:
			if name != "" && v.Name.Local != name {
				return xml.StartElement{}, errors.Errorf("xmlrpc nextStartToken: unknown tag %q", v.Name.Local)
			}
			return v, nil
		case xml.CharData:
			// NOTE: There's often CharData in here we don't care about.
		default:
			return xml.StartElement{}, errors.Errorf("xmlrpc nextStartToken: invalid token type: %T", t)
		}
	}
}

func nextEndToken(d *xml.Decoder, name string) (xml.EndElement, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return xml.EndElement{}, err
		}

		switch v := t.(type) {
		case xml.EndElement:
			if name != "" && v.Name.Local != name {
				return xml.EndElement{}, errors.Errorf("xmlrpc nextEndToken: unknown tag %q", v.Name.Local)
			}
			return v, nil
		case xml.CharData:
			// NOTE: There's often CharData in here we don't care about.
		default:
			return xml.EndElement{}, errors.Errorf("xmlrpc nextEndToken: invalid token type: %T", t)
		}
	}
}

func nextCharToken(d *xml.Decoder) (xml.CharData, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return xml.CharData{}, err
		}

		switch v := t.(type) {
		case xml.CharData:
			return v, nil
		default:
			return xml.CharData{}, errors.Errorf("xmlrpc nextCharToken: invalid token type: %T", t)
		}
	}
}

func nextProcInst(d *xml.Decoder) (xml.ProcInst, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return xml.ProcInst{}, err
		}

		switch v := t.(type) {
		case xml.ProcInst:
			return v, nil
		default:
			return xml.ProcInst{}, errors.Errorf("xmlrpc nextProcInst: invalid token type: %T", t)
		}
	}
}

func nextStartOrEndToken(d *xml.Decoder, start string, end string) (xml.Token, error) {
	for {
		t, err := d.Token()
		if err != nil {
			return t, err
		}

		switch v := t.(type) {
		case xml.StartElement:
			if start != "" && v.Name.Local != start {
				return t, errors.Errorf("xmlrpc nextStartOrEndToken: unknown start tag %q", v.Name.Local)
			}
			return v, nil
		case xml.EndElement:
			if end != "" && v.Name.Local != end {
				return t, errors.Errorf("xmlrpc nextStartOrEndToken: unknown end tag %q", v.Name.Local)
			}
			return v, nil
		case xml.CharData:
			// NOTE: There's often CharData in here we don't care about.
		default:
			return t, errors.Errorf("xmlrpc nextStartOrEndToken: invalid token type: %T", t)
		}
	}
}
