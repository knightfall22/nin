package transmission

import (
	"fmt"
	"io"
)

type Parser struct {
	rd io.Reader
	wr io.Writer
}

func NewParser(input io.ReadWriter) *Parser {
	return &Parser{
		rd: input,
		wr: input,
	}
}

func (p *Parser) Process() error {
	msg, err := DeserializeMessageFromReader(p.rd)
	if err != nil {
		return err
	}

	fmt.Printf("Message: %+v\n", msg)

	// _, err = p.wr.Write(byt)
	// if err != nil {
	// 	return err
	// }

	return nil
}
