package blocktemplate

import "io"

type BlockHashingBlob struct {
	BlockHeader  BlockHeader
	TreeRootHash CryptoHash
	TxHashSize   uint64
}

func (b BlockHashingBlob) Pack(writer io.Writer) error {
	err := b.BlockHeader.Pack(writer)
	if err != nil {
		return err
	}
	err = b.TreeRootHash.Pack(writer)
	if err != nil {
		return err
	}
	err = PackVarInt(writer, b.TxHashSize)
	if err != nil {
		return err
	}
	return nil
}

func (b *BlockHashingBlob) UnPack(reader io.Reader) error {
	err := b.BlockHeader.UnPack(reader)
	if err != nil {
		return err
	}
	err = b.TreeRootHash.UnPack(reader)
	if err != nil {
		return err
	}
	b.TxHashSize, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	return nil
}
