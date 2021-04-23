package blocktemplate

import "io"

type BlockTemplateBlob struct {
	Block Block
}

func (b BlockTemplateBlob) Pack(writer io.Writer) error {
	err := b.Block.Pack(writer)
	if err != nil {
		return err
	}
	return nil
}

func (b *BlockTemplateBlob) UnPack(reader io.Reader) error {
	err := b.Block.UnPack(reader)
	if err != nil {
		return err
	}
	return nil
}
