// Package format provides XML pretty-printing.
//
// Write reads a single XML document from r and writes it to w with consistent
// two-space indentation. The XML declaration, comments, processing
// instructions, and CDATA sections are preserved. The output always ends with
// a newline.
package format

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Write reads an XML document from r, pretty-prints it, and writes to w.
func Write(w io.Writer, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}

	var buf bytes.Buffer
	dec := xml.NewDecoder(bytes.NewReader(data))

	// depth tracks element nesting for indentation.
	depth := 0

	// frame records what kind of content an open element has seen so far.
	// hasChildElems is set when at least one child StartElement was seen;
	// when true the closing tag is placed on its own indented line.
	// When only text was seen the closing tag stays inline.
	type frame struct{ hasChildElems bool }
	var stack []frame

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("parse xml: %w", err)
		}

		switch t := tok.(type) {
		case xml.ProcInst:
			// XML declaration or processing instruction.
			if buf.Len() > 0 {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat("  ", depth))
			}
			buf.WriteString("<?")
			buf.WriteString(t.Target)
			if len(t.Inst) > 0 {
				buf.WriteByte(' ')
				buf.Write(t.Inst)
			}
			buf.WriteString("?>")

		case xml.Comment:
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat("  ", depth))
			buf.WriteString("<!--")
			buf.Write(t)
			buf.WriteString("-->")

		case xml.Directive:
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat("  ", depth))
			buf.WriteString("<!")
			buf.Write(t)
			buf.WriteByte('>')

		case xml.StartElement:
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat("  ", depth))
			buf.WriteByte('<')
			buf.WriteString(elemName(t.Name))
			for _, a := range t.Attr {
				buf.WriteByte(' ')
				buf.WriteString(attrName(a.Name))
				buf.WriteString(`="`)
				xml.EscapeText(&buf, []byte(a.Value)) //nolint:errcheck
				buf.WriteByte('"')
			}
			buf.WriteByte('>')
			// Mark the parent as having a child element.
			if len(stack) > 0 {
				stack[len(stack)-1].hasChildElems = true
			}
			stack = append(stack, frame{})
			depth++

		case xml.EndElement:
			depth--
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			// Only indent the closing tag when child elements were present;
			// text-only elements keep the closing tag on the same line.
			if top.hasChildElems {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat("  ", depth))
			}
			buf.WriteString("</")
			buf.WriteString(elemName(t.Name))
			buf.WriteByte('>')

		case xml.CharData:
			trimmed := strings.TrimSpace(string(t))
			if trimmed == "" {
				// Pure whitespace between tags — skip; we provide our own indentation.
				continue
			}
			buf.WriteString(trimmed)
		}
	}

	buf.WriteByte('\n')
	_, err = w.Write(buf.Bytes())
	return err
}

// elemName returns the qualified name of an XML element.
func elemName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

// attrName returns the qualified name of an XML attribute.
func attrName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}
