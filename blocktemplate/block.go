package blocktemplate

import "io"
import btc_utility "github.com/mutalisk999/bitcoin-lib/src/utility"

type BlockHeader struct {
	MajorVersion uint8
	MinorVersion uint8
	Timestamp    uint64
	PrevId       CryptoHash
	Nonce        uint32
}

func (b BlockHeader) Pack(writer io.Writer) error {
	err := PackVarInt(writer, uint64(b.MajorVersion))
	if err != nil {
		return err
	}
	err = PackVarInt(writer, uint64(b.MinorVersion))
	if err != nil {
		return err
	}
	err = PackVarInt(writer, b.Timestamp)
	if err != nil {
		return err
	}
	err = b.PrevId.Pack(writer)
	if err != nil {
		return err
	}
	err = packU32(writer, b.Nonce)
	if err != nil {
		return err
	}
	return nil
}

func (b *BlockHeader) UnPack(reader io.Reader) error {
	r, err := UnPackVarInt(reader)
	if err != nil {
		return err
	}
	b.MajorVersion = uint8(r)
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	b.MinorVersion = uint8(r)
	r, err = UnPackVarInt(reader)
	if err != nil {
		return err
	}
	b.Timestamp = r
	err = b.PrevId.UnPack(reader)
	if err != nil {
		return err
	}
	u32, err := unpackU32(reader)
	if err != nil {
		return err
	}
	b.Nonce = u32
	return nil
}

type Block struct {
	BlockHeader BlockHeader
	MinerTx     MinerTransaction
	TxHashes    []CryptoHash
}

func (b Block) Pack(writer io.Writer) error {
	err := b.BlockHeader.Pack(writer)
	if err != nil {
		return err
	}
	err = b.MinerTx.Pack(writer)
	if err != nil {
		return err
	}
	err = PackVarInt(writer, uint64(len(b.TxHashes)))
	if err != nil {
		return err
	}
	for i := 0; i < len(b.TxHashes); i++ {
		err = b.TxHashes[i].Pack(writer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Block) UnPack(reader io.Reader) error {
	err := b.BlockHeader.UnPack(reader)
	if err != nil {
		return err
	}
	err = b.MinerTx.UnPack(reader)
	if err != nil {
		return err
	}
	r, err := UnPackVarInt(reader)
	if err != nil {
		return err
	}
	b.TxHashes = make([]CryptoHash, int(r))
	for i := 0; i < int(r); i++ {
		err = b.TxHashes[i].UnPack(reader)
		if err != nil {
			return err
		}
	}
	return nil
}

func packU32(writer io.Writer, data32 uint32) error {
	bytes := btc_utility.DumpUint32ToBytes(btc_utility.ConvertUint32ToLittleEndian(data32))
	btc_utility.Assert(len(bytes) == 4, "incorrect bytes length, not 4")
	_, err := writer.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func unpackU32(reader io.Reader) (uint32, error) {
	var bytes [4]byte
	_, err := reader.Read(bytes[0:4])
	if err != nil {
		return 0, err
	}
	u32 := btc_utility.ConvertUint32FromLittleEndian(btc_utility.LoadUint32FromBytes(bytes[0:4]))
	return u32, nil
}
